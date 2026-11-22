package browser

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
)

func checkUploadResponse(t *testing.T, response *httptest.ResponseRecorder, status int, code string) string {
	t.Helper()
	got := decodeResponse(t, response)
	if response.Code != status || got.Code != code {
		t.Fatalf("响应 %d %+v，预期 %d %q", response.Code, got, status, code)
	}
	if status == http.StatusAccepted {
		if !got.OK || got.JobID == "" {
			t.Fatalf("缺少任务编号：%+v", got)
		}
	} else if got.OK || got.JobID != "" || got.Error == "" {
		t.Fatalf("错误响应不完整：%+v", got)
	}
	return got.JobID
}

type pausedUploadReader struct {
	io.Reader
	waiting chan struct{}
}

func (reader *pausedUploadReader) Read(data []byte) (int, error) {
	if reader.waiting != nil {
		close(reader.waiting)
		reader.waiting = nil
	}
	return reader.Reader.Read(data)
}

func TestUploadBudgetCombinesReceivingAndQueuedFiles(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		name := "总量超限"
		if interrupted {
			name = "上传断开"
		}
		t.Run(name, func(t *testing.T) {
			releaseImport := make(chan struct{})
			blockImport, unblockImport := context.WithCancel(t.Context())
			defer unblockImport()
			h := New(Options{Import: func(ctx context.Context, request Request) error {
				if request.File == "" {
					close(releaseImport)
					<-blockImport.Done()
				}
				return nil
			}}).(*handler)
			h.jobs.uploadLimit = 12

			blocker := request(t, h, "POST", "/api/v1/import", `{"url":"https://example.com/block.png"}`, extensionOrigin)
			checkUploadResponse(t, blocker, http.StatusAccepted, "")
			select {
			case <-releaseImport:
			case <-time.After(2 * time.Second):
				t.Fatal("阻塞任务没有开始")
			}
			queuedID := checkUploadResponse(t, upload(t, h, "", map[string]string{"queued.png": "123456"}), http.StatusAccepted, "")

			const content = "YxZ9"
			stream := uploadRequest(t, "", map[string]string{"receiving.png": content + "!"})
			body, err := io.ReadAll(stream.Body)
			if err != nil {
				t.Fatal(err)
			}
			split := bytes.Index(body, []byte(content)) + len(content)
			head, tail := body[:split], body[split:]
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()
			waiting := make(chan struct{})
			stream.Body = struct {
				io.Reader
				io.Closer
			}{io.MultiReader(bytes.NewReader(head), &pausedUploadReader{Reader: reader, waiting: waiting}), reader}
			stream.ContentLength = -1
			stream.TransferEncoding = []string{"chunked"}
			recorder := httptest.NewRecorder()
			received := make(chan struct{})
			go func() {
				h.ServeHTTP(recorder, stream)
				close(received)
			}()
			select {
			case <-waiting:
			case <-time.After(2 * time.Second):
				t.Fatal("分块上传未读到暂停处")
			}
			over := upload(t, h, "", map[string]string{"over.png": "123"})
			checkUploadResponse(t, over, http.StatusServiceUnavailable, codeBusy)
			if !strings.Contains(decodeResponse(t, over).Error, "Upload storage is full") {
				t.Fatal("响应未说明上传容量耗尽")
			}
			exactID := checkUploadResponse(t, upload(t, h, "", map[string]string{"exact.png": "12"}), http.StatusAccepted, "")
			if interrupted {
				writer.CloseWithError(io.ErrUnexpectedEOF)
			} else {
				go func() {
					defer writer.Close()
					io.Copy(writer, bytes.NewReader(tail))
				}()
			}
			select {
			case <-received:
			case <-time.After(2 * time.Second):
				t.Fatal("上传未结束")
			}
			if interrupted {
				checkUploadResponse(t, recorder, http.StatusBadRequest, codeBadRequest)
			} else {
				checkUploadResponse(t, recorder, http.StatusServiceUnavailable, codeBusy)
			}
			retryID := checkUploadResponse(t, upload(t, h, "", map[string]string{"retry.png": "1234"}), http.StatusAccepted, "")
			unblockImport()
			for _, id := range []string{queuedID, exactID, retryID} {
				if got := waitJob(t, h, id); got.State != jobDone {
					t.Fatalf("任务结果 %+v", got)
				}
			}
		})
	}
}

func TestUploadBudgetReleasesFinishedJobs(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code string
	}{
		{name: "导入成功"},
		{name: "导入失败", err: errors.New("磁盘写入失败"), code: codeInternal},
		{name: "取消导入", err: i18n.NewError("import_canceled"), code: codeCanceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			release, cancel := context.WithCancel(t.Context())
			defer cancel()
			var tempFile string
			h := New(Options{Import: func(_ context.Context, request Request) error {
				if request.FileName == "first.png" {
					tempFile = request.File
					<-release.Done()
					return tc.err
				}
				return nil
			}}).(*handler)
			h.jobs.uploadLimit = 4
			first := checkUploadResponse(t, upload(t, h, "", map[string]string{"first.png": "1234"}), http.StatusAccepted, "")
			checkUploadResponse(t, upload(t, h, "", map[string]string{"full.png": "1"}), http.StatusServiceUnavailable, codeBusy)
			cancel()
			want := jobDone
			if tc.err != nil {
				want = jobFailed
			}
			if got := waitJob(t, h, first); got.State != want || got.Code != tc.code {
				t.Fatalf("任务结果 %+v", got)
			}
			if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
				t.Fatalf("临时文件未清理：%v", err)
			}
			retry := checkUploadResponse(t, upload(t, h, "", map[string]string{"retry.png": "1234"}), http.StatusAccepted, "")
			waitJob(t, h, retry)
		})
	}
}

func TestUploadBudgetReleasesCanceledQueuedJob(t *testing.T) {
	active, stop := context.WithCancel(t.Context())
	defer stop()
	finish, release := context.WithCancel(t.Context())
	defer release()
	started := make(chan struct{})
	h := New(Options{Import: func(ctx context.Context, request Request) error {
		if request.FileName == "active.png" {
			close(started)
			<-ctx.Done()
			<-finish.Done()
			return ctx.Err()
		}
		return nil
	}}).(*handler)
	h.jobs.uploadLimit = 4
	submit := func(name string) string {
		req := uploadRequest(t, "", map[string]string{name: "12"})
		client, disconnect := context.WithCancel(context.WithValue(active, serverContextKey{}, active))
		defer disconnect()
		recorder := httptest.NewRecorder()
		h.ServeHTTP(recorder, req.WithContext(client))
		return checkUploadResponse(t, recorder, http.StatusAccepted, "")
	}
	activeID := submit("active.png")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("导入没有开始")
	}
	queuedID := submit("queued.png")
	queued := request(t, h, "GET", "/api/v1/import/"+queuedID, "", extensionOrigin)
	if got := decodeResponse(t, queued); got.Job == nil || got.Job.State != jobQueued {
		t.Fatalf("客户端断开后排队任务状态变化：%+v", got)
	}
	stop()
	if got := waitJob(t, h, queuedID); got.State != jobFailed || got.Code != codeCanceled {
		t.Fatalf("取消排队任务的结果 %+v", got)
	}
	checkUploadResponse(t, upload(t, h, "", map[string]string{"over.png": "123"}), http.StatusServiceUnavailable, codeBusy)
	retry := checkUploadResponse(t, upload(t, h, "", map[string]string{"retry.png": "12"}), http.StatusAccepted, "")
	release()
	if got := waitJob(t, h, activeID); got.State != jobFailed || got.Code != codeCanceled {
		t.Fatalf("取消运行中任务的结果 %+v", got)
	}
	waitJob(t, h, retry)
	full := checkUploadResponse(t, upload(t, h, "", map[string]string{"full.png": "1234"}), http.StatusAccepted, "")
	waitJob(t, h, full)
}

func TestUploadBudgetReleasesRejectedRequests(t *testing.T) {
	active, stop := context.WithCancel(t.Context())
	defer stop()
	blocker := make(chan struct{})
	h := New(Options{Import: func(ctx context.Context, request Request) error {
		if request.URL == "https://example.com/block.png" {
			close(blocker)
			<-ctx.Done()
		}
		return ctx.Err()
	}}).(*handler)
	h.jobs.uploadLimit = 4
	files := map[string]string{"example.png": "1234"}
	checkUploadResponse(t, upload(t, h, `{`, files), http.StatusBadRequest, codeBadRequest)

	h.jobs.submit(active, Request{URL: "https://example.com/block.png"}, nil)
	select {
	case <-blocker:
	case <-time.After(2 * time.Second):
		t.Fatal("阻塞任务没有开始")
	}
	var last *job
	for range maxPendingJobs {
		var ok bool
		last, ok = h.jobs.submit(active, Request{}, nil)
		if !ok {
			t.Fatal("队列提前达到上限")
		}
	}
	full := upload(t, h, "", files)
	checkUploadResponse(t, full, http.StatusServiceUnavailable, codeBusy)
	if got := decodeResponse(t, full).Error; got != "Too many imports are waiting." {
		t.Fatalf("未到达队列满的拒绝路径：%q", got)
	}
	stop()
	waitJob(t, h, last.id)
	retry := checkUploadResponse(t, upload(t, h, "", files), http.StatusAccepted, "")
	waitJob(t, h, retry)
}

func TestUploadFileLimitKeepsTooLarge(t *testing.T) {
	upload := uploadFile{bytes: MaxUploadSize}
	written, err := upload.Write([]byte("1"))
	status, code, message := uploadReadError(err)
	if written != 0 || status != http.StatusRequestEntityTooLarge || code != codeTooLarge || message == "" {
		t.Fatalf("超过单文件上限后写入 %d 字节，响应 %d %q %q", written, status, code, message)
	}
}
