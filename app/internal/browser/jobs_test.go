package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
)

const extensionOrigin = "chrome-extension://bdholohelelbeepbpdapldbibidacaoo"

func waitJob(t *testing.T, h http.Handler, id string) jobView {
	t.Helper()
	queue := h.(*handler).jobs
	queue.mu.Lock()
	job := queue.jobs[id]
	queue.mu.Unlock()
	if job == nil {
		t.Fatalf("找不到任务 %q", id)
	}
	select {
	case <-job.done:
	case <-time.After(2 * time.Second):
		t.Fatalf("任务 %q 没有结束", id)
	}
	w := request(t, h, "GET", "/api/v1/import/"+id, "", extensionOrigin)
	got := decodeResponse(t, w)
	if w.Code != http.StatusOK || got.Job == nil {
		t.Fatalf("查询任务 %d %+v", w.Code, got)
	}
	return *got.Job
}

func TestStatusCapabilities(t *testing.T) {
	w := request(t, New(Options{Version: "1.2.3"}), "GET", "/api/v1/status", "", extensionOrigin)
	got := decodeResponse(t, w)
	if got.Version != "1.2.3" || got.Capabilities == nil || !got.Capabilities.Jobs || !got.Capabilities.Upload || got.Capabilities.MaxUploadBytes != MaxUploadSize {
		t.Fatalf("%+v", got)
	}
}

func TestAsyncImportRunsInOrderAndReportsResult(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	var order []string
	h := New(Options{Import: func(ctx context.Context, r Request) error {
		<-release
		mu.Lock()
		order = append(order, r.URL)
		mu.Unlock()
		if strings.Contains(r.URL, "bad") {
			return i18n.NewError("error_download_status", 403)
		}
		return nil
	}})
	var ids []string
	for _, name := range []string{"a", "bad", "c"} {
		body := fmt.Sprintf(`{"url":"https://example.com/%s.png","kind":"image","tags":["x"],"batchId":"b1","index":%d,"total":3}`, name, len(ids))
		w := request(t, h, "POST", "/api/v1/import", body, extensionOrigin)
		got := decodeResponse(t, w)
		if w.Code != http.StatusAccepted || !got.OK || got.JobID == "" {
			t.Fatalf("%d %+v", w.Code, got)
		}
		ids = append(ids, got.JobID)
	}
	// 还没放行时任务是排队或进行中，不是结束。
	if w := request(t, h, "GET", "/api/v1/import/"+ids[2], "", extensionOrigin); decodeResponse(t, w).Job.State != jobQueued {
		t.Fatal("第三个任务应该还在排队")
	}
	close(release)
	if v := waitJob(t, h, ids[0]); v.State != jobDone || v.Code != "" {
		t.Fatalf("%+v", v)
	}
	if v := waitJob(t, h, ids[1]); v.State != jobFailed || v.Code != codeRemoteRefused || v.Error == "" {
		t.Fatalf("%+v", v)
	}
	waitJob(t, h, ids[2])
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(order, ",") != "https://example.com/a.png,https://example.com/bad.png,https://example.com/c.png" {
		t.Fatalf("顺序 %v", order)
	}
	if w := request(t, h, "GET", "/api/v1/import/unknown", "", extensionOrigin); w.Code != http.StatusNotFound || decodeResponse(t, w).Code != codeNotFound {
		t.Fatalf("未知任务 %d", w.Code)
	}
}

func TestImportValidatesNewFields(t *testing.T) {
	h := New(Options{Import: func(context.Context, Request) error { return nil }})
	for _, body := range []string{
		`{"url":"https://example.com/a.png","kind":"audio"}`,
		`{"url":"https://example.com/a.png","index":3,"total":3}`,
		`{"url":"https://example.com/a.png","referer":"ftp://example.com"}`,
		`{"url":"https://example.com/a.png","tags":["` + strings.Repeat("长", 101) + `"]}`,
	} {
		if w := request(t, h, "POST", "/api/v1/import", body, extensionOrigin); w.Code != http.StatusBadRequest || decodeResponse(t, w).Code != codeBadRequest {
			t.Fatalf("%s: %d", body, w.Code)
		}
	}
}

func upload(t *testing.T, h http.Handler, meta string, files map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, uploadRequest(t, meta, files))
	return w
}

func uploadRequest(t *testing.T, meta string, files map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if meta != "" {
		if err := writer.WriteField("meta", meta); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range files {
		part, err := writer.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/v1/import/file", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Origin", extensionOrigin)
	return r
}

func TestUploadPassesFileAndRemovesTemp(t *testing.T) {
	var got Request
	var content []byte
	h := New(Options{Import: func(_ context.Context, r Request) error {
		got = r
		var err error
		content, err = os.ReadFile(r.File)
		return err
	}})
	w := upload(t, h, `{"pageUrl":"https://example.com/page","title":"标题","tags":["猫"]}`, map[string]string{"截图.png": "png-data"})
	resp := decodeResponse(t, w)
	if w.Code != http.StatusAccepted || resp.JobID == "" {
		t.Fatalf("%d %+v", w.Code, resp)
	}
	if v := waitJob(t, h, resp.JobID); v.State != jobDone {
		t.Fatalf("%+v", v)
	}
	if got.FileName != "截图.png" || got.PageURL != "https://example.com/page" || got.Title != "标题" || len(got.Tags) != 1 || string(content) != "png-data" {
		t.Fatalf("%+v %q", got, content)
	}
	if _, err := os.Stat(got.File); !os.IsNotExist(err) {
		t.Fatalf("临时文件没删：%v", err)
	}
}

func TestUploadRejectsBadInput(t *testing.T) {
	calls := 0
	h := New(Options{Import: func(context.Context, Request) error { calls++; return nil }})
	for name, tc := range map[string]struct {
		meta  string
		files map[string]string
	}{
		"没有文件":   {meta: `{}`},
		"两个文件":   {meta: `{}`, files: map[string]string{"a.png": "1", "b.png": "2"}},
		"元数据坏了":  {meta: `{`, files: map[string]string{"a.png": "1"}},
		"来源不是网址": {meta: `{"pageUrl":"file:///tmp"}`, files: map[string]string{"a.png": "1"}},
	} {
		if w := upload(t, h, tc.meta, tc.files); w.Code != http.StatusBadRequest || decodeResponse(t, w).Code != codeBadRequest {
			t.Fatalf("%s: %d", name, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/api/v1/import/file", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非 multipart 返回 %d", w.Code)
	}
	if calls != 0 {
		t.Fatalf("坏请求触发了 %d 次导入", calls)
	}
}

func TestClassify(t *testing.T) {
	for _, tc := range []struct {
		err    error
		code   string
		status int
	}{
		{i18n.NewError("library_required"), codeNoLibrary, http.StatusConflict},
		{i18n.NewError("import_canceled"), codeCanceled, http.StatusConflict},
		{i18n.WrapError(errors.New("dial"), "error_download_failed", "example.com"), codeDownload, http.StatusBadGateway},
		{errors.Join(errors.New("x"), i18n.NewError("error_download_format", "text/html")), codeUnsupported, http.StatusUnsupportedMediaType},
		{fmt.Errorf("wrap: %w", i18n.NewError("error_download_too_large", 200)), codeTooLarge, http.StatusRequestEntityTooLarge},
		{context.Canceled, codeCanceled, http.StatusConflict},
		{errors.New("磁盘满了"), codeInternal, http.StatusInternalServerError},
	} {
		if code, status := classify(tc.err); code != tc.code || status != tc.status {
			t.Fatalf("%v: %s %d", tc.err, code, status)
		}
	}
}

func TestLegacyImportReturnsCode(t *testing.T) {
	h := New(Options{Import: func(context.Context, Request) error { return i18n.NewError("library_required") }})
	w := request(t, h, "POST", "/api/v1/import/image", `{"url":"https://example.com/a.png"}`, extensionOrigin)
	var got response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusConflict || got.Code != codeNoLibrary || got.Error == "" {
		t.Fatalf("%d %+v %v", w.Code, got, err)
	}
}
