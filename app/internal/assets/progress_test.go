package assets

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
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
	for i := range 5 {
		fixtureCopy(t, "gif", filepath.Join(input, fmt.Sprintf("existing-%03d.gif", i)))
	}
	for i := range 32 {
		img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		img.SetNRGBA(0, 0, color.NRGBA{R: uint8(i + 1), A: 255})
		path := filepath.Join(input, fmt.Sprintf("fresh-%03d.png", i))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, img); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 83 {
		fixtureCopy(t, "jpg", filepath.Join(input, fmt.Sprintf("pending-%03d.jpg", i)))
	}
	releaseLastPNG, releaseJPEG := make(chan struct{}), make(chan struct{})
	var pngOnce, jpegOnce sync.Once
	releaseAll := func() { pngOnce.Do(func() { close(releaseLastPNG) }); jpegOnce.Do(func() { close(releaseJPEG) }) }
	pngDecoded := make(chan struct{}, 31)
	s.prepareImage = func(path string) (image.Image, map[string]string, int, string, error) {
		if filepath.Ext(path) == ".jpg" {
			<-releaseJPEG
		}
		img, values, orientation, profile, err := PrepareImage(path)
		if err == nil && filepath.Ext(path) == ".png" {
			pixel := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
			if pixel.R == 32 {
				<-releaseLastPNG
			} else {
				pngDecoded <- struct{}{}
			}
		}
		return img, values, orientation, profile, err
	}
	events := make(chan model.ImportProgress, 64)
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
	for range 31 {
		select {
		case <-pngDecoded:
		case <-time.After(3 * time.Second):
			t.Fatal("PNG解码没有完成")
		}
	}
	// 留住最后一张 PNG 和 JPEG，越过节流窗口后只放行 PNG。
	timer := time.NewTimer(110 * time.Millisecond)
	<-timer.C
	pngOnce.Do(func() { close(releaseLastPNG) })
	found := false
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for !found {
		select {
		case p := <-events:
			if p.Stage == "importing" {
				if p.Processed != p.Imported+p.Skipped {
					t.Fatalf("处理计数不相等：%+v", p)
				}
				if p.Processed == 37 {
					if p.Total != 120 || p.Imported != 32 || p.Skipped != 5 {
						t.Fatalf("处理中计数不符合状态表：%+v", p)
					}
					found = true
				}
			}
		case err := <-done:
			finished = true
			t.Fatalf("还未完成JPEG却结束：%v", err)
		case <-timeout.C:
			t.Fatal("PNG已准备完成，但没有37/120、导入32、重复5的中途进度")
		}
	}
	var persisted int
	if err := lib.DB.QueryRowContext(ctx, "SELECT count(*) FROM assets").Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 1 {
		t.Fatalf("尚未收尾却写入了%d条素材", persisted)
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
		if p.Processed != p.Imported+p.Skipped {
			t.Fatalf("处理计数不相等：%+v", p)
		}
		final = p
	}
	if final.Stage != "finishing" || final.Total != 120 || final.Processed != 120 || final.Imported != 33 || final.Skipped != 87 {
		t.Fatalf("最终计数错误：%+v", final)
	}
}

func TestImportCommitFailureResetsFinalResultAndRollsBackPreparedAssets(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	if _, err := lib.DB.ExecContext(ctx, `CREATE TABLE commit_guard(asset_id TEXT REFERENCES assets(id),folder_id TEXT REFERENCES folders(id) DEFERRABLE INITIALLY DEFERRED);
 CREATE TRIGGER reject_commit AFTER INSERT ON assets BEGIN INSERT INTO commit_guard(asset_id,folder_id) VALUES(NEW.id,'missing-folder'); END`); err != nil {
		t.Fatal(err)
	}
	var events []model.ImportProgress
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample.png")}}, func(p model.ImportProgress) { events = append(events, p) })
	if err == nil {
		t.Fatal("延迟外键约束应在提交时失败")
	}
	prepared := false
	for _, p := range events {
		if p.Stage == "importing" || p.Stage == "finishing" {
			if p.Processed != p.Imported+p.Skipped {
				t.Fatalf("处理计数不相等：%+v", p)
			}
			if p.Imported == 1 {
				prepared = true
			}
		}
	}
	if !prepared {
		t.Fatal("没有报告素材准备完成的真实进度")
	}
	if result.Imported != 0 || result.Skipped != 0 || result.Restored != 0 {
		t.Fatalf("失败的最终结果没有清零：%+v", result)
	}
	if len(snapshot(t, context.Background(), lib).Assets) != 0 || len(storedFiles(t, lib.Root)) != 0 {
		t.Fatal("失败提交留下素材或文件")
	}
}
