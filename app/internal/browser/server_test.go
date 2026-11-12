package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func request(t *testing.T, handler http.Handler, method, path, body, origin string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) response {
	t.Helper()
	var got response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatal("响应不是 JSON")
	}
	return got
}

func TestStatus(t *testing.T) {
	w := request(t, New(nil), "GET", "/api/v1/status", "", "")
	got := decodeResponse(t, w)
	if w.Code != http.StatusOK || !got.OK || got.Status != "ready" || got.Error != "" {
		t.Fatalf("%d: %+v", w.Code, got)
	}
}

func TestImportPayload(t *testing.T) {
	for _, body := range []string{
		`{"url":"https://example.com/image.png"}`,
		`{"url":"https://example.com/image.png","pageUrl":"https://example.com/page","playFeedback":false}`,
		`{"url":"https://example.com/image.png","pageUrl":"https://example.com/page","playFeedback":true,"futureField":1}`,
	} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			h := New(func(ctx context.Context, imageURL, pageURL string) error {
				calls++
				if ctx == nil || imageURL != "https://example.com/image.png" {
					t.Fatalf("%v %q", ctx, imageURL)
				}
				if strings.Contains(body, "pageUrl") && pageURL != "https://example.com/page" {
					t.Fatalf("来源链接为 %q", pageURL)
				}
				return nil
			})
			w := request(t, h, "POST", "/api/v1/import/image", body, "chrome-extension://example/")
			if got := decodeResponse(t, w); w.Code != http.StatusOK || !got.OK || calls != 1 {
				t.Fatalf("%d %+v, 调用 %d 次", w.Code, got, calls)
			}
			if w.Header().Get("Access-Control-Allow-Origin") != "chrome-extension://example/" {
				t.Fatal("缺少扩展来源响应头")
			}
		})
	}
}

func TestOriginValidation(t *testing.T) {
	for _, origin := range []string{"", "chrome-extension://example", "chrome-extension://example/"} {
		w := request(t, New(nil), "GET", "/api/v1/status", "", origin)
		if w.Code != http.StatusOK {
			t.Fatalf("允许来源 %q 返回 %d", origin, w.Code)
		}
	}
	for _, origin := range []string{"https://example.com", "null", "moz-extension://example", "https://chrome-extension://example"} {
		w := request(t, New(nil), "GET", "/api/v1/status", "", origin)
		if got := decodeResponse(t, w); w.Code != http.StatusForbidden || got.OK || got.Error == "" {
			t.Fatalf("拒绝来源 %q: %d %+v", origin, w.Code, got)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("拒绝来源获得了 CORS 授权")
		}
	}
	r := httptest.NewRequest("GET", "/api/v1/status", nil)
	r.Header.Add("Origin", "chrome-extension://example")
	r.Header.Add("Origin", "https://example.com")
	w := httptest.NewRecorder()
	New(nil).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatal("多个 Origin 被接受")
	}
}

func TestRoutesAndPreflight(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		code         int
		allow        string
	}{
		{"POST", "/api/v1/status", http.StatusMethodNotAllowed, "GET, OPTIONS"},
		{"GET", "/api/v1/import/image", http.StatusMethodNotAllowed, "POST, OPTIONS"},
		{"DELETE", "/api/v1/import/image", http.StatusMethodNotAllowed, "POST, OPTIONS"},
		{"GET", "/missing", http.StatusNotFound, ""},
		{"OPTIONS", "/api/v1/import/image", http.StatusNoContent, "POST, OPTIONS"},
		{"OPTIONS", "/api/v1/status", http.StatusNoContent, "GET, OPTIONS"},
	} {
		w := request(t, New(nil), tc.method, tc.path, "", "chrome-extension://example")
		if w.Code != tc.code || w.Header().Get("Allow") != tc.allow {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Header().Get("Allow"))
		}
		if tc.method == "OPTIONS" {
			if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" || w.Header().Get("Access-Control-Allow-Methods") != tc.allow || w.Body.Len() != 0 {
				t.Fatal("预检响应不完整")
			}
		} else if got := decodeResponse(t, w); got.OK || got.Error == "" {
			t.Fatalf("错误响应为 %+v", got)
		}
	}
}

func TestRejectsInvalidPayloadWithoutImport(t *testing.T) {
	calls := 0
	h := New(func(context.Context, string, string) error { calls++; return nil })
	for _, body := range []string{
		"", "null", "[]", "{}", "{", `{"url":"https://example.com/image.png"} {}`,
		`{"url":1}`, `{"url":""}`, `{"url":"relative.png"}`,
		`{"url":"file:///tmp/image.png"}`, `{"url":"javascript:alert(1)"}`,
		`{"url":"https:///image.png"}`, `{"url":"https://example.com/image.png","pageUrl":"file:///tmp/page"}`,
		`{"url":"https://example.com/image.png","playFeedback":"yes"}`,
	} {
		w := request(t, h, "POST", "/api/v1/import/image", body, "")
		if got := decodeResponse(t, w); w.Code != http.StatusBadRequest || got.OK || got.Error == "" {
			t.Fatalf("%q: %d %+v", body, w.Code, got)
		}
	}
	if calls != 0 {
		t.Fatalf("非法请求触发了 %d 次导入", calls)
	}
}

func TestBodyLimitIncludingChunkedRequest(t *testing.T) {
	base := `{"url":"https://example.com/image.png"}`
	calls := 0
	h := New(func(context.Context, string, string) error { calls++; return nil })
	for _, chunked := range []bool{false, true} {
		for _, length := range []int{MaxBodySize, MaxBodySize + 1} {
			r := httptest.NewRequest("POST", "/api/v1/import/image", strings.NewReader(base+strings.Repeat(" ", length-len(base))))
			if chunked {
				r.ContentLength = -1
				r.TransferEncoding = []string{"chunked"}
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want := http.StatusOK
			if length > MaxBodySize {
				want = http.StatusRequestEntityTooLarge
			}
			if w.Code != want {
				t.Fatalf("长度 %d，分块 %t: %d，想要 %d", length, chunked, w.Code, want)
			}
		}
	}
	if calls != 2 {
		t.Fatalf("超限请求触发导入，共 %d 次", calls)
	}
}

type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (brokenBody) Close() error             { return nil }

func TestBodyReadFailure(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/import/image", nil)
	r.Body = brokenBody{}
	w := httptest.NewRecorder()
	New(nil).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("读取失败返回 %d", w.Code)
	}
}

func TestImportFailureAndNotReady(t *testing.T) {
	body := `{"url":"https://example.com/image.png"}`
	w := request(t, New(nil), "POST", "/api/v1/import/image", body, "")
	if got := decodeResponse(t, w); w.Code != http.StatusServiceUnavailable || got.OK || got.Error == "" {
		t.Fatalf("%d %+v", w.Code, got)
	}
	w = request(t, New(func(context.Context, string, string) error { return errors.New("没有打开图库") }), "POST", "/api/v1/import/image", body, "")
	if got := decodeResponse(t, w); w.Code != http.StatusInternalServerError || got.OK || got.Error != "没有打开图库" {
		t.Fatalf("%d %+v", w.Code, got)
	}
}

func TestRequestLogOmitsImageURLsAndQueries(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	h := New(func(context.Context, string, string) error {
		return errors.New("download https://example.com/image.png?secret=private-token failed")
	})
	request(t, h, "POST", "/api/v1/import/image?request-secret=private-token", `{"url":"https://example.com/image.png?secret=private-token"}`, "")
	if strings.Contains(logs.String(), "private-token") || strings.Contains(logs.String(), "example.com") {
		t.Fatalf("日志暴露链接: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "status=500") {
		t.Fatalf("日志没有记录错误结果: %s", logs.String())
	}
}

func TestServerStopsAndCancelsImport(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	served := make(chan error, 1)
	go func() {
		served <- serve(ctx, listener, New(func(ctx context.Context, _, _ string) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}))
	}()
	clientDone := make(chan error, 1)
	go func() {
		client := http.Client{Timeout: 3 * time.Second}
		res, err := client.Post("http://"+listener.Addr().String()+"/api/v1/import/image", "application/json", strings.NewReader(`{"url":"https://example.com/image.png"}`))
		if err == nil {
			_, err = io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
		clientDone <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("服务未接到导入请求")
	}
	cancel()
	select {
	case err := <-served:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("服务没有停止")
	}
	if err := <-clientDone; err != nil {
		t.Fatal(err)
	}
	connection, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err == nil {
		connection.Close()
		t.Fatal("服务停止后仍可连接")
	}
}
