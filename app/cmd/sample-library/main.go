package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("sample-library", flag.ContinueOnError)
	flags.SetOutput(out)
	path := flags.String("path", "", "新图库完整路径，必须以 .momento 结尾")
	count := flags.Int("count", 14, "生成的素材数量")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *path == "" || !strings.HasSuffix(strings.ToLower(*path), ".momento") {
		return fmt.Errorf("请用 -path 指定新的 .momento 图库路径")
	}
	if *count < 0 || flags.NArg() != 0 {
		return fmt.Errorf("count 不能小于 0，且不接受额外参数")
	}
	return generate(ctx, *path, *count, out)
}

func generate(ctx context.Context, path string, count int, out io.Writer) (err error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	lib, err := library.Create(ctx, path, name)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil && ctx.Err() != nil {
			err = errors.Join(ctx.Err(), err)
		}
		err = errors.Join(err, lib.Close())
		if err != nil {
			err = errors.Join(err, os.RemoveAll(lib.Root))
		}
	}()
	tx, err := lib.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	insert, err := tx.PrepareContext(ctx, `INSERT INTO assets
		(id,name,original_file_name,extension,media_type,size,width,height,relative_path,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer insert.Close()
	insertColor, err := tx.PrepareContext(ctx, "INSERT INTO asset_colors(asset_id,hex,proportion,family,sort_index) VALUES(?,?,?,?,?)")
	if err != nil {
		return err
	}
	defer insertColor.Close()
	start, reported := time.Now(), time.Now()
	baseDate := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	pngEncoder := png.Encoder{CompressionLevel: png.BestSpeed}
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		img := sampleImage(i, count <= 14)
		ext, media := "jpg", "image/jpeg"
		var original bytes.Buffer
		if count <= 14 && i >= 10 {
			ext, media = "png", "image/png"
			err = pngEncoder.Encode(&original, img)
		} else {
			err = jpeg.Encode(&original, img, &jpeg.Options{Quality: 85})
			if err == nil {
				// JPEG 注释保留样本编号，保证每份有效原文件内容唯一。
				data := original.Bytes()
				comment := fmt.Appendf(nil, "Momento sample %d", i+1)
				withComment := make([]byte, 0, len(data)+len(comment)+4)
				withComment = append(withComment, data[:2]...)
				withComment = append(withComment, 0xff, 0xfe)
				withComment = binary.BigEndian.AppendUint16(withComment, uint16(len(comment)+2))
				withComment = append(withComment, comment...)
				withComment = append(withComment, data[2:]...)
				original.Reset()
				original.Write(withComment)
			}
		}
		if err != nil {
			return err
		}
		hash := sha256.Sum256(original.Bytes())
		id := hex.EncodeToString(hash[:])
		rel := filepath.ToSlash(filepath.Join("assets", id[:2], id+"."+ext))
		if err = os.MkdirAll(filepath.Dir(filepath.Join(lib.Root, rel)), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(lib.Root, rel), original.Bytes(), 0600); err != nil {
			return err
		}
		var thumbnail bytes.Buffer
		if err = pngEncoder.Encode(&thumbnail, img); err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(lib.Root, "thumbnails", id+".png"), thumbnail.Bytes(), 0600); err != nil {
			return err
		}
		stamp := baseDate.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano)
		assetName := fmt.Sprintf("Sample %06d", i+1)
		if _, err = insert.ExecContext(ctx, id, assetName, assetName+"."+ext, ext, media, original.Len(), img.Bounds().Dx(), img.Bounds().Dy(), rel, stamp, stamp); err != nil {
			return err
		}
		for index, swatch := range assets.Palette(img) {
			if _, err = insertColor.ExecContext(ctx, id, swatch.Hex, swatch.Proportion, swatch.Family, index); err != nil {
				return err
			}
		}
		if time.Since(reported) >= time.Second {
			fmt.Fprintf(out, "已生成 %d / %d\n", i+1, count)
			reported = time.Now()
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	fmt.Fprintf(out, "已生成 %d 个素材：%s（%s）\n", count, lib.Root, time.Since(start).Round(time.Millisecond))
	return nil
}
