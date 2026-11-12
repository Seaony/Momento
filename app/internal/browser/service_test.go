package browser

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServiceToggleAndRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := make(chan Status, 8)
	service := newService(ctx, "127.0.0.1:0", New(nil), func(status Status) { changes <- status })
	defer service.SetEnabled(false)
	if got := service.Status(); got.State != Disabled || got.Err != nil {
		t.Fatalf("初始状态错误：%+v", got)
	}
	for i := 0; i < 2; i++ {
		if err := service.SetEnabled(true); err != nil {
			t.Fatal(err)
		}
		running := <-changes
		if running.State != Running || running.Err != nil {
			t.Fatalf("启动状态错误：%+v", running)
		}
		client := http.Client{Timeout: time.Second}
		res, err := client.Get("http://" + running.Address + "/api/v1/status")
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil || res.StatusCode != http.StatusOK || string(body) != "{\"ok\":true,\"status\":\"ready\"}\n" {
			t.Fatalf("HTTP契约变化：%d %q %v", res.StatusCode, body, readErr)
		}
		if err := service.SetEnabled(true); err != nil || service.Status().Address != running.Address {
			t.Fatalf("重复开启改变了服务：%+v %v", service.Status(), err)
		}
		if err := service.SetEnabled(false); err != nil {
			t.Fatal(err)
		}
		if got := <-changes; got.State != Disabled || got.Err != nil {
			t.Fatalf("关闭状态错误：%+v", got)
		}
		assertDisconnected(t, running.Address)
	}
}

func TestServicePortOccupiedAndRetry(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := newService(ctx, listener.Addr().String(), New(nil), nil)
	defer service.SetEnabled(false)
	if err := service.SetEnabled(true); err == nil {
		t.Fatal("端口被占用却报告启动成功")
	}
	if got := service.Status(); got.State != PortOccupied || got.Err == nil {
		t.Fatalf("占用状态错误：%+v", got)
	}
	if err := service.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if got := service.Status(); got.State != Disabled || got.Err != nil {
		t.Fatalf("关闭后残留错误：%+v", got)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := service.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if got := service.Status(); got.State != Running || got.Err != nil {
		t.Fatalf("释放端口后没有恢复：%+v", got)
	}
}

func TestServiceStopCancelsImport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	service := newService(ctx, "127.0.0.1:0", New(func(ctx context.Context, _, _ string) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}), nil)
	defer service.SetEnabled(false)
	if err := service.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	address := service.Status().Address
	clientDone := make(chan error, 1)
	go func() {
		client := http.Client{Timeout: 3 * time.Second}
		res, err := client.Post("http://"+address+"/api/v1/import/image", "application/json", strings.NewReader(`{"url":"https://example.com/image.png"}`))
		if err == nil {
			_, err = io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
		clientDone <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("导入请求未到达")
	}
	if err := service.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if err := <-clientDone; err != nil {
		t.Fatal(err)
	}
	assertDisconnected(t, address)
}

func TestServiceContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := make(chan Status, 4)
	var service *Service
	service = newService(ctx, "127.0.0.1:0", New(nil), func(status Status) {
		_ = service.Status()
		changes <- status
	})
	if err := service.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	address := (<-changes).Address
	cancel()
	select {
	case got := <-changes:
		if got.State != Disabled || service.Status().State != Disabled {
			t.Fatalf("应用退出后未关闭：%+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("应用退出后没有通知关闭")
	}
	assertDisconnected(t, address)
	if err := service.SetEnabled(true); !errors.Is(err, context.Canceled) {
		t.Fatalf("应用退出后仍可重新启动：%v", err)
	}
}

func TestServiceStartupErrorIsNotPortOccupied(t *testing.T) {
	service := newService(context.Background(), "invalid-address", New(nil), nil)
	if err := service.SetEnabled(true); err == nil {
		t.Fatal("非法地址却报告启动成功")
	}
	if got := service.Status(); got.State != Failed || got.Err == nil {
		t.Fatalf("其他错误被误报为端口占用：%+v", got)
	}
}

func assertDisconnected(t *testing.T, address string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp4", address, time.Second)
	if err == nil {
		conn.Close()
		t.Fatal("服务关闭后仍能连接")
	}
}
