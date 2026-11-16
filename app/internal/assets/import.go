package assets

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func hashReader(ctx context.Context, r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, contextReader{ctx: ctx, r: r}); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type candidate struct {
	path      string
	folders   []string
	staged    string
	hash      string
	size      int64
	extension string
	createdAt time.Time
}
type prepared struct {
	asset               model.Asset
	original, thumbnail string
}

func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tif", ".tiff", ".bmp", ".heic", ".heif", ".avif":
		return true
	}
	return false
}
func nowText() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func scan(ctx context.Context, paths []string) ([]candidate, error) {
	var out []candidate
	seen := map[string]bool{}
	add := func(path string, folders []string) error {
		if !Supported(path) || strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if seen[absolute] {
			return nil
		}
		seen[absolute] = true
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
		out = append(out, candidate{path: absolute, folders: folders, extension: ext})
		return nil
	}
	for _, root := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if info.Mode().IsRegular() {
				if err = add(root, nil); err != nil {
					return nil, err
				}
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if path == root {
				return nil
			}
			if strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				if strings.EqualFold(filepath.Ext(entry.Name()), ".momento") {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			var folders []string
			if rel != "." {
				folders = strings.Split(rel, string(filepath.Separator))
			}
			return add(path, folders)
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out, nil
}

type progressReporter struct {
	callback func(model.ImportProgress)
	last     time.Time
}

func (p *progressReporter) send(v model.ImportProgress, force bool) {
	if p.callback != nil && (force || time.Since(p.last) >= 100*time.Millisecond) {
		p.callback(v)
		p.last = time.Now()
	}
}

func parallel(ctx context.Context, count int, work func(context.Context, int) error, completed ...func(int)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	workers := min(count, min(max(runtime.NumCPU(), 2), 6))
	if workers == 0 {
		return ctx.Err()
	}
	jobs := make(chan int)
	finished := make(chan int, workers)
	var wg sync.WaitGroup
	var once sync.Once
	var first error
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				if ctx.Err() != nil {
					continue
				}
				if err := work(ctx, i); err != nil {
					once.Do(func() { first = err; cancel() })
					continue
				}
				finished <- i
			}
		})
	}
	go func() {
		defer close(jobs)
		for i := range count {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(finished) }()
	for i := range finished {
		if len(completed) > 0 {
			completed[0](i)
		}
	}
	if first != nil {
		return first
	}
	return ctx.Err()
}

func stageHash(ctx context.Context, c *candidate, stage string, index int) (err error) {
	in, err := os.Open(c.path)
	if err != nil {
		return sourceError{err}
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return sourceError{err}
	}
	c.createdAt, _ = sourceCreatedAt(info)
	c.staged = filepath.Join(stage, fmt.Sprintf("source-%d.%s", index, c.extension))
	out, err := os.OpenFile(c.staged, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	h := sha256.New()
	c.size, err = io.Copy(io.MultiWriter(out, h), contextReader{ctx: ctx, r: sourceReader{in}})
	err = errors.Join(err, out.Close())
	if err != nil {
		return err
	}
	if err = os.Chtimes(c.staged, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	c.hash = hex.EncodeToString(h.Sum(nil))
	return nil
}

// sourceError 标记读源文件时出的错：只跳过这一张，不影响整批。
type sourceError struct{ err error }

func (e sourceError) Error() string { return e.err.Error() }
func (e sourceError) Unwrap() error { return e.err }

type sourceReader struct{ r io.Reader }

func (r sourceReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if err != nil && err != io.EOF {
		err = sourceError{err}
	}
	return n, err
}

type existingAsset struct {
	trashed  bool
	relative string
	rotation int
}

// Import 整批导入。单张图片读不了、解不开或太大时只跳过这一张，记进 result.Errors；
// 取消、数据库和写图库文件出错时整批回滚。
func (s *Service) Import(ctx context.Context, request model.ImportRequest, callback func(model.ImportProgress)) (result model.ImportResult, err error) {
	if !s.importing.TryLock() {
		return result, ErrImportBusy
	}
	defer s.importing.Unlock()
	s.mutation.Lock()
	defer s.mutation.Unlock()
	slog.Info("开始导入", "paths", len(request.Paths))
	defer func() {
		if err != nil {
			slog.Error("导入失败", "error", err)
		} else {
			slog.Info("导入完成", "imported", result.Imported, "restored", result.Restored, "skipped", result.Skipped, "failed", len(result.Errors))
		}
	}()
	progress := progressReporter{callback: callback}
	progress.send(model.ImportProgress{Stage: "scanning"}, true)
	candidates, err := scan(ctx, request.Paths)
	if err != nil {
		return result, err
	}
	total := len(candidates)
	progress.send(model.ImportProgress{Stage: "scanning", Total: total}, true)
	if total == 0 {
		progress.send(model.ImportProgress{Stage: "finishing"}, true)
		return result, nil
	}
	stage, err := os.MkdirTemp(s.root, ".import-")
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(stage)) }()
	// 每个候选文件自己的失败原因，非空的这张不导入。
	fileErrs := make([]error, total)
	fail := func(i int, cause error) {
		name := filepath.Base(candidates[i].path)
		slog.Warn("导入时跳过一张图片", "file", name, "error", cause)
		// 解码错误里带的是暂存文件名（source-3.heic），用户看不懂；去掉这一层，只留原因，文件名由外层给出。
		var decode *i18n.LocalizedError
		if errors.As(cause, &decode) && decode.Key == "error_decode_image" && decode.Cause != nil {
			var limit *i18n.LocalizedError
			if errors.As(decode.Cause, &limit) {
				cause = decode.Cause
			} else {
				cause = i18n.WrapError(decode.Cause, "error_image_unreadable")
			}
		}
		fileErrs[i] = i18n.WrapError(cause, "error_import_read", name)
	}
	hashed := 0
	if err = parallel(ctx, total, func(ctx context.Context, i int) error {
		stageErr := stageHash(ctx, &candidates[i], stage, i)
		var source sourceError
		if errors.As(stageErr, &source) && ctx.Err() == nil {
			fail(i, source.err)
			return nil
		}
		return stageErr
	}, func(i int) {
		hashed++
		progress.send(model.ImportProgress{Stage: "scanning", Total: total, Processed: hashed, CurrentFile: filepath.Base(candidates[i].path)}, false)
	}); err != nil {
		return result, err
	}
	existing := map[string]existingAsset{}
	rows, err := s.db.QueryContext(ctx, "SELECT id,trashed,relative_path,rotation FROM assets")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var id string
		var a existingAsset
		if err = rows.Scan(&id, &a.trashed, &a.relative, &a.rotation); err != nil {
			rows.Close()
			return result, err
		}
		existing[id] = a
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return result, err
	}
	groups := map[string][]int{}
	var hashes []string
	for i, c := range candidates {
		if fileErrs[i] != nil {
			continue
		}
		if len(groups[c.hash]) == 0 {
			hashes = append(hashes, c.hash)
		}
		groups[c.hash] = append(groups[c.hash], i)
	}
	failGroup := func(hash string, cause error) {
		for _, i := range groups[hash] {
			fail(i, cause)
		}
	}
	// 已在图库里的先核对原图：缺了就用这次的文件补回，内容不对只跳过这一组。
	repair := map[string]bool{}
	var toPrepare []string
	for _, hash := range hashes {
		old, exists := existing[hash]
		if !exists {
			toPrepare = append(toPrepare, hash)
			continue
		}
		missing, checkErr := s.storedCopyMissing(ctx, old.relative, hash)
		if err = ctx.Err(); err != nil {
			return result, err
		}
		if checkErr != nil {
			failGroup(hash, checkErr)
			continue
		}
		if missing {
			slog.Warn("图库里缺少原图，用这次导入的文件补回", "id", hash, "path", old.relative)
			repair[hash] = true
			toPrepare = append(toPrepare, hash)
		}
	}
	keywordsByHash := map[string][]string{}
	if request.ImportKeywords {
		for _, hash := range hashes {
			if err = ctx.Err(); err != nil {
				return result, err
			}
			c := candidates[groups[hash][0]]
			if fileErrs[groups[hash][0]] != nil {
				continue
			}
			// 关键词读不出来不影响图片本身，跳过这张的关键词即可。
			words, readErr := ReadKeywords(c.staged)
			if readErr != nil {
				slog.Warn("读取图片关键词失败，跳过这张的关键词", "file", filepath.Base(c.path), "error", readErr)
				continue
			}
			keywordsByHash[hash] = words
		}
	}
	processed, skipped, failed := 0, 0, 0
	for i := range candidates {
		if fileErrs[i] != nil {
			processed++
			failed++
		}
	}
	for _, hash := range hashes {
		old, exists := existing[hash]
		if !exists || repair[hash] || fileErrs[groups[hash][0]] != nil {
			continue
		}
		processed += len(groups[hash])
		skipped += len(groups[hash])
		if old.trashed {
			skipped--
		}
	}
	report := func(stage, file string, force bool) {
		progress.send(model.ImportProgress{Stage: stage, Total: total, Processed: processed, Imported: processed - skipped - failed, Skipped: skipped, CurrentFile: file}, force)
	}
	ready := make([]prepared, len(toPrepare))
	prepareErrs := make([]error, len(toPrepare))
	// 进度里的导入数统计已准备好的非重复素材，最终结果仍以整批提交为准。
	report("importing", "", true)
	if err = parallel(ctx, len(toPrepare), func(ctx context.Context, j int) error {
		hash := toPrepare[j]
		c := candidates[groups[hash][0]]
		img, metadata, orientation, profile, err := s.prepareImage(c.staged)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			prepareErrs[j] = err
			return nil
		}
		thumb := filepath.Join(stage, c.hash+".png")
		if repair[hash] {
			// 补回的缩略图要带上图库里已有的旋转。
			rotated, err := RotateImage(img, existing[hash].rotation)
			if err != nil {
				return err
			}
			if err = WriteThumbnail(thumb, rotated); err != nil {
				return err
			}
			ready[j] = prepared{original: c.staged, thumbnail: thumb}
			return nil
		}
		if !c.createdAt.IsZero() {
			metadata["fileCreatedAt"] = c.createdAt.UTC().Format(time.RFC3339Nano)
		} else {
			delete(metadata, "fileCreatedAt")
		}
		if err = WriteThumbnail(thumb, img); err != nil {
			return err
		}
		name := filepath.Base(c.path)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		now := time.Now().UTC()
		ready[j] = prepared{asset: model.Asset{ID: c.hash, Name: name, OriginalFileName: filepath.Base(c.path), Extension: c.extension, MediaType: mediaType(c.extension), Size: c.size, Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), Orientation: orientation, ColorProfile: profile, EXIF: metadata, SourceURL: request.SourceURL, RelativePath: filepath.ToSlash(filepath.Join("assets", c.hash[:2], c.hash+"."+c.extension)), CreatedAt: now, UpdatedAt: now, Colors: Palette(img)}, original: c.staged, thumbnail: thumb}
		return nil
	}, func(j int) {
		hash := toPrepare[j]
		count := len(groups[hash])
		processed += count
		if prepareErrs[j] != nil {
			failed += count
		} else {
			skipped += count - 1
			if repair[hash] && !existing[hash].trashed {
				skipped++
			}
		}
		report("importing", filepath.Base(candidates[groups[hash][0]].path), false)
	}); err != nil {
		return result, err
	}
	byHash := make(map[string]prepared, len(ready))
	for j, hash := range toPrepare {
		if prepareErrs[j] != nil {
			failGroup(hash, prepareErrs[j])
			continue
		}
		byHash[hash] = ready[j]
	}
	report("importing", "", true)
	report("finishing", "", true)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	committed := false
	var installed []string
	defer func() {
		if !committed {
			_ = tx.Rollback()
			for _, path := range installed {
				err = errors.Join(err, removeIfExists(path))
			}
			result = model.ImportResult{}
		}
	}()
	folderCache := map[string]string{}
	repaired := map[string]bool{}
	resultIDs := map[string]bool{}
	for i, c := range candidates {
		if fileErrs[i] != nil {
			continue
		}
		var trashed bool
		var source, relative string
		lookupErr := tx.QueryRowContext(ctx, "SELECT trashed,source_url,relative_path FROM assets WHERE id=?", c.hash).Scan(&trashed, &source, &relative)
		switch {
		case errors.Is(lookupErr, sql.ErrNoRows):
			p, ok := byHash[c.hash]
			if !ok || repair[c.hash] {
				return result, i18n.NewError("error_import_record_changed", c.hash)
			}
			if err = insertAsset(ctx, tx, p.asset); err != nil {
				return result, err
			}
			for _, file := range []struct {
				source, relative string
				original         bool
			}{{p.original, p.asset.RelativePath, true}, {p.thumbnail, filepath.Join("thumbnails", c.hash+".png"), false}} {
				dst, placed, installErr := s.installFile(ctx, file.source, file.relative, c.hash, file.original)
				if placed {
					installed = append(installed, dst)
				}
				if installErr != nil {
					return result, installErr
				}
			}
			result.Imported++
		case lookupErr != nil:
			return result, lookupErr
		default:
			if repair[c.hash] && !repaired[c.hash] {
				// 记录本来就在，补回的文件即使这批回滚也留着。
				p := byHash[c.hash]
				for _, file := range []struct {
					source, relative string
					original         bool
				}{{p.original, relative, true}, {p.thumbnail, filepath.Join("thumbnails", c.hash+".png"), false}} {
					if _, _, err = s.installFile(ctx, file.source, file.relative, c.hash, file.original); err != nil {
						return result, err
					}
				}
				repaired[c.hash] = true
				slog.Info("已补回图库缺失的原图和缩略图", "id", c.hash, "path", relative)
			}
			if trashed {
				result.Imported++
				result.Restored++
			} else {
				result.Skipped++
			}
			if trashed || (source == "" && request.SourceURL != "") {
				if _, err = tx.ExecContext(ctx, "UPDATE assets SET trashed=0,source_url=CASE WHEN source_url='' THEN ? ELSE source_url END,updated_at=? WHERE id=?", request.SourceURL, nowText(), c.hash); err != nil {
					return result, err
				}
			}
		}
		if !resultIDs[c.hash] {
			result.AssetIDs = append(result.AssetIDs, c.hash)
			resultIDs[c.hash] = true
		}
		for _, tagID := range request.TagIDs {
			if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO asset_tags(asset_id,tag_id) VALUES(?,?)", c.hash, tagID); err != nil {
				return result, err
			}
		}
		if err = attachKeywords(ctx, tx, c.hash, keywordsByHash[c.hash]); err != nil {
			return result, err
		}
		parent := request.FolderID
		for _, name := range c.folders {
			key := parent + "\x00" + name
			id, ok := folderCache[key]
			if !ok {
				id, err = ensureFolder(ctx, tx, name, parent)
				if err != nil {
					return result, err
				}
				folderCache[key] = id
			}
			parent = id
		}
		if parent != "" {
			if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO asset_folders(asset_id,folder_id) VALUES(?,?)", c.hash, parent); err != nil {
				return result, err
			}
		}
		report("finishing", filepath.Base(c.path), false)
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	committed = true
	for _, fileErr := range fileErrs {
		if fileErr != nil {
			result.Errors = append(result.Errors, fileErr)
		}
	}
	progress.send(model.ImportProgress{Stage: "finishing", Total: total, Processed: total, Imported: total - result.Skipped - len(result.Errors), Skipped: result.Skipped}, true)
	return result, nil
}

// installFile 把暂存文件放进图库，placed 表示这次新放进去的（回滚时要删）。
// 原图已存在且内容一致，说明上次导入中途中断留下了它，直接沿用；缩略图可再生，已存在就覆盖。
func (s *Service) installFile(ctx context.Context, staged, relative, hash string, original bool) (dst string, placed bool, err error) {
	dst, err = s.storageDestination(relative)
	if err != nil {
		return "", false, err
	}
	info, statErr := os.Lstat(dst)
	switch {
	case os.IsNotExist(statErr):
		if err = publishFile(staged, dst); err != nil {
			return dst, false, err
		}
		return dst, true, nil
	case statErr != nil:
		return dst, false, statErr
	case !info.Mode().IsRegular():
		return dst, false, i18n.NewError("error_import_unregistered_file", relative)
	case !original:
		if err = os.Rename(staged, dst); err != nil {
			return dst, false, err
		}
		return dst, true, nil
	}
	same, err := fileHashEquals(ctx, dst, hash)
	if err != nil {
		return dst, false, err
	}
	if !same {
		return dst, false, i18n.NewError("error_import_unregistered_file", relative)
	}
	slog.Info("图库里已有内容相同的原图，直接沿用", "path", relative)
	return dst, false, nil
}

func mediaType(ext string) string {
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "tif", "tiff":
		return "image/tiff"
	case "heic", "heif":
		return "image/heic"
	default:
		return "image/" + ext
	}
}

func insertAsset(ctx context.Context, tx *sql.Tx, a model.Asset) error {
	metadata, err := json.Marshal(a.EXIF)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO assets(id,name,original_file_name,extension,media_type,size,width,height,orientation,color_profile,exif_json,note,source_url,relative_path,favorite,trashed,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, a.ID, a.Name, a.OriginalFileName, a.Extension, a.MediaType, a.Size, a.Width, a.Height, a.Orientation, a.ColorProfile, string(metadata), a.Note, a.SourceURL, a.RelativePath, a.Favorite, a.Trashed, a.CreatedAt.Format(time.RFC3339Nano), a.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return replaceColors(ctx, tx, a.ID, a.Colors)
}
func replaceColors(ctx context.Context, tx *sql.Tx, id string, colors []model.Color) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM asset_colors WHERE asset_id=?", id); err != nil {
		return err
	}
	for i, c := range colors {
		if _, err := tx.ExecContext(ctx, "INSERT INTO asset_colors(asset_id,hex,proportion,family,sort_index) VALUES(?,?,?,?,?)", id, c.Hex, c.Proportion, c.Family, i); err != nil {
			return err
		}
	}
	return nil
}
func ensureFolder(ctx context.Context, tx *sql.Tx, name, parent string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, "SELECT id FROM folders WHERE name=? AND COALESCE(parent_id,'')=? ORDER BY sort_index,id LIMIT 1", name, parent).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id = rand.Text()
	var parentValue any
	if parent != "" {
		parentValue = parent
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO folders(id,name,parent_id,sort_index) VALUES(?,?,?,(SELECT COALESCE(MAX(sort_index)+1,0) FROM folders WHERE COALESCE(parent_id,'')=?))", id, name, parentValue, parent)
	return id, err
}
