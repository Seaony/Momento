package assets

import (
	"context"
	"image"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func TestImportReportsActualProcessingBeforeCommit(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "gif")
	input := t.TempDir()
	fixtureCopy(t, "gif", filepath.Join(input, "existing-1.gif"))
	fixtureCopy(t, "gif", filepath.Join(input, "existing-2.gif"))
	fixtureCopy(t, "png", filepath.Join(input, "fresh-1.png"))
	fixtureCopy(t, "png", filepath.Join(input, "fresh-2.png"))
	fixtureCopy(t, "jpg", filepath.Join(input, "fresh-3.jpg"))
	releasePNG, releaseJPEG := make(chan struct{}), make(chan struct{})
	var pngOnce, jpegOnce sync.Once
	releaseAll := func() { pngOnce.Do(func() { close(releasePNG) }); jpegOnce.Do(func() { close(releaseJPEG) }) }
	started := make(chan string, 2)
	s.prepareImage = func(path string) (image.Image, map[string]string, int, string, error) {
		extension := filepath.Ext(path)
		started <- extension
		if extension == ".png" {
			<-releasePNG
		} else {
			<-releaseJPEG
		}
		return PrepareImage(path)
	}
	events := make(chan model.ImportProgress, 32)
	done := make(chan error, 1)
	finished := false
	defer func() {
		releaseAll()
		if !finished {
			<-done
		}
	}()
	go func() {
		_, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, func(p model.ImportProgress) { events <- p })
		done <- err
	}()
	for range 2 {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("解码没有开始")
		}
	}
	// 两个工作者都停在可控的解码边界，越过节流窗口后只放行 PNG。
	timer := time.NewTimer(110 * time.Millisecond)
	<-timer.C
	pngOnce.Do(func() { close(releasePNG) })
	found := false
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for !found {
		select {
		case p := <-events:
			if p.Stage == "importing" && p.Processed == 4 {
				if p.Total != 5 || p.Skipped != 3 || p.Imported != 0 {
					t.Fatalf("处理中计数错误：%+v", p)
				}
				found = true
			}
		case err := <-done:
			finished = true
			t.Fatalf("还未完成 JPEG 却结束：%v", err)
		case <-timeout.C:
			t.Fatal("PNG 已处理完但进度仍停在零，没有中途处理进度")
		}
	}
	var persisted int
	if err := lib.DB.QueryRowContext(ctx, "SELECT count(*) FROM assets").Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 1 {
		t.Fatalf("尚未收尾却写入了 %d 条素材", persisted)
	}
	jpegOnce.Do(func() { close(releaseJPEG) })
	err := <-done
	finished = true
	if err != nil {
		t.Fatal(err)
	}
	close(events)
	var final model.ImportProgress
	for p := range events {
		if p.Imported != 0 && p.Stage != "finishing" {
			t.Fatalf("提交前宣称已导入：%+v", p)
		}
		final = p
	}
	if final.Stage != "finishing" || final.Total != 5 || final.Processed != 5 || final.Imported != 2 || final.Skipped != 3 {
		t.Fatalf("最终计数错误：%+v", final)
	}
}

func TestImportCommitFailureDoesNotReportImportedAssets(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	if _, err := lib.DB.ExecContext(ctx, `CREATE TABLE commit_guard(asset_id TEXT REFERENCES assets(id),folder_id TEXT REFERENCES folders(id) DEFERRABLE INITIALLY DEFERRED);
 CREATE TRIGGER reject_commit AFTER INSERT ON assets BEGIN INSERT INTO commit_guard(asset_id,folder_id) VALUES(NEW.id,'missing-folder'); END`); err != nil {
		t.Fatal(err)
	}
	var events []model.ImportProgress
	_, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample.png")}}, func(p model.ImportProgress) { events = append(events, p) })
	if err == nil {
		t.Fatal("延迟外键约束应在提交时失败")
	}
	for _, p := range events {
		if p.Imported != 0 {
			t.Fatalf("提交失败却报告已导入：%+v", p)
		}
	}
	if len(snapshot(t, context.Background(), lib).Assets) != 0 || len(storedFiles(t, lib.Root)) != 0 {
		t.Fatal("失败提交留下素材或文件")
	}
}
