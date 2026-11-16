package library

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// 造一个接近真实的大图库：每张带常见 EXIF、5 个主色，部分在文件夹里、带标签。
func seedBenchmarkLibrary(tb testing.TB, count int) *Library {
	tb.Helper()
	ctx := context.Background()
	l, err := Create(ctx, filepath.Join(tb.TempDir(), "基准.momento"), "基准")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := l.Close(); err != nil {
			tb.Error(err)
		}
	})
	tx, err := l.DB.BeginTx(ctx, nil)
	if err != nil {
		tb.Fatal(err)
	}
	defer tx.Rollback()
	exec := func(query string, args ...any) {
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			tb.Fatal(err)
		}
	}
	for i := range 20 {
		exec("INSERT INTO folders(id,name,sort_index) VALUES(?,?,?)", fmt.Sprintf("folder-%02d", i), fmt.Sprintf("文件夹 %d", i), i)
		exec("INSERT INTO tags(id,name,normalized_name) VALUES(?,?,?)", fmt.Sprintf("tag-%02d", i), fmt.Sprintf("标签 %d", i), fmt.Sprintf("标签 %d", i))
	}
	exif, err := json.Marshal(map[string]string{
		"cameraMake": "SONY", "cameraModel": "ILCE-7M4", "lensModel": "FE 35mm F1.4 GM", "exposureTime": "0.004",
		"focalLength": "35", "fNumber": "2.8", "isoSpeedRatings": "200", "flashFired": "false", "exposureProgram": "3",
		"meteringMode": "5", "whiteBalance": "0", "dpiWidth": "300", "dpiHeight": "300", "pixelWidth": "6000",
		"pixelHeight": "4000", "colorModel": "RGB", "profileName": "Display P3", "contentCreatedAt": "2026-01-02T09:05:00Z",
		"fileModifiedAt": "2026-01-02T09:05:00Z",
	})
	if err != nil {
		tb.Fatal(err)
	}
	insert, err := tx.PrepareContext(ctx, "INSERT INTO assets(id,name,original_file_name,extension,media_type,size,width,height,color_profile,exif_json,relative_path,created_at,updated_at,last_used_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		tb.Fatal(err)
	}
	defer insert.Close()
	color, err := tx.PrepareContext(ctx, "INSERT INTO asset_colors(asset_id,hex,proportion,family,sort_index) VALUES(?,?,?,?,?)")
	if err != nil {
		tb.Fatal(err)
	}
	defer color.Close()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	families := []string{"red", "orange", "yellow", "green", "blue", "purple", "gray"}
	for i := range count {
		// 编号打散后当 id，模拟哈希 id：插入顺序和 id 顺序不同。
		id := fmt.Sprintf("%016x%048d", uint64(i)*0x9e3779b97f4a7c15, i)
		stamp := base.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano)
		used := ""
		if i%7 == 0 {
			used = stamp
		}
		if _, err := insert.ExecContext(ctx, id, fmt.Sprintf("素材 %06d", i), fmt.Sprintf("IMG_%06d.jpg", i), "jpg", "image/jpeg", 1000+i, 6000, 4000, "Display P3", string(exif), "assets/"+id[:2]+"/"+id+".jpg", stamp, stamp, used); err != nil {
			tb.Fatal(err)
		}
		for j := range 5 {
			if _, err := color.ExecContext(ctx, id, fmt.Sprintf("#%06x", (i*5+j)&0xffffff), 0.2, families[(i+j)%len(families)], j); err != nil {
				tb.Fatal(err)
			}
		}
		if i%5 == 0 {
			exec("INSERT INTO asset_folders(asset_id,folder_id) VALUES(?,?)", id, fmt.Sprintf("folder-%02d", i%20))
		}
		if i%3 == 0 {
			exec("INSERT INTO asset_tags(asset_id,tag_id) VALUES(?,?),(?,?)", id, fmt.Sprintf("tag-%02d", (i+7)%20), id, fmt.Sprintf("tag-%02d", i%20))
		}
	}
	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}
	return l
}

// 打开 10 万张图库时读快照的耗时、分配，以及读完后快照本身常驻的堆内存（retained-MB）。
func BenchmarkSnapshot100K(b *testing.B) {
	l := seedBenchmarkLibrary(b, 100_000)
	var retained float64
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		b.StartTimer()
		s, err := l.Snapshot(b.Context())
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&after)
		retained = float64(after.HeapAlloc-before.HeapAlloc) / (1 << 20)
		runtime.KeepAlive(s)
		b.StartTimer()
	}
	b.ReportMetric(retained, "retained-MB")
}
