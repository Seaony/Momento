package assets

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestResidentAndNormalReleaseWakeConcurrently(t *testing.T) {
	for range 100 {
		budget := newMemoryBudget(100)
		releaseResident, ok := budget.tryReserveResident(20)
		if !ok {
			t.Fatal("常驻额度分配失败")
		}
		releaseNormal, err := budget.reserve(context.Background(), 80)
		if err != nil {
			t.Fatal(err)
		}
		started := time.Now()
		acquired := make(chan int64, 1)
		go func() {
			granted, err := budget.acquire(context.Background(), 30)
			if err != nil {
				acquired <- -1
				return
			}
			acquired <- granted
		}()
		budget.mu.Lock()
		waiting := len(budget.waiting)
		budget.mu.Unlock()
		for waiting != 1 && time.Since(started) < time.Second {
			time.Sleep(time.Microsecond)
			budget.mu.Lock()
			waiting = len(budget.waiting)
			budget.mu.Unlock()
		}
		if waiting != 1 {
			t.Fatalf("后续请求没有排队：%d", waiting)
		}
		var releases sync.WaitGroup
		releases.Add(2)
		go func() { defer releases.Done(); releaseResident() }()
		go func() { defer releases.Done(); releaseNormal() }()
		releases.Wait()
		granted := <-acquired
		if granted != 30 {
			t.Fatalf("排队请求拿到 %d", granted)
		}
		budget.release(granted)
		budget.mu.Lock()
		used, waiting := budget.used, len(budget.waiting)
		budget.mu.Unlock()
		if used != 0 || waiting != 0 {
			t.Fatalf("归还后状态不对：used=%d waiting=%d", used, waiting)
		}
	}
}
