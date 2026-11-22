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
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
)

func hashReader(ctx context.Context, r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, contextReader{ctx: ctx, r: r}); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashFile 是给导入状态核对这类外部入口用的内容哈希；图库内文件身份用素材 ID，不再重复算。
func HashFile(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return hashReader(ctx, file)
}

type candidate struct {
	path      string
	folders   []string
	staged    string
	hash      string
	size      int64
	extension string
	createdAt time.Time
	motion    *motionSource // 配成实况照片时的那段视频
	pairErr   error
}
type prepared struct {
	asset               model.Asset
	original, thumbnail string // thumbnail 是暂存的缩略图，后缀 .jpg 或 .png
}

func Supported(path string) bool {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tif", ".tiff", ".bmp", ".heic", ".heif", ".avif":
		return true
	default:
		return isVideo(ext) || media.IsSystemImageExtension(ext)
	}
}
func nowText() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// scanned 是展开拖入路径后的结果。
type scanned struct {
	candidates []candidate
	// unsupported 是格式不支持的文件数（含文件夹里的），example 是其中一个文件名，用来提示用户。
	unsupported int
	example     string
	// direct 是用户直接拖进来、格式不支持的文件（绝对路径），和图片一起拖进来时要逐个报出来。
	direct []string
}

// scan 展开拖入的文件和文件夹。链接（含指向文件夹的）会跟过去扫，同一个真实文件夹只扫一次，防止链接绕圈。
// macOS 访达的「替身」不是链接，纯 Go 解析不了，按不支持的文件计数。
func scan(ctx context.Context, paths []string) (scanned, error) {
	var out scanned
	seen := map[string]bool{}
	visited := map[string]bool{}
	add := func(path string, folders []string) error {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if seen[absolute] {
			return nil
		}
		seen[absolute] = true
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
		out.candidates = append(out.candidates, candidate{path: absolute, folders: folders, extension: ext})
		return nil
	}
	unsupported := func(path string) {
		out.unsupported++
		if out.example == "" {
			out.example = filepath.Base(path)
		}
	}
	var walk func(dir string, folders []string) error
	walk = func(dir string, folders []string) error {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return err
		}
		if visited[real] {
			slog.Warn("导入时跳过重复或绕圈的文件夹", "path", dir)
			return nil
		}
		visited[real] = true
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			path := filepath.Join(dir, name)
			mode := entry.Type()
			if mode&os.ModeSymlink != 0 {
				info, err := os.Stat(path)
				if err != nil {
					slog.Warn("导入时跳过失效的链接", "path", path, "error", err)
					continue
				}
				mode = info.Mode().Type()
			}
			switch {
			case mode.IsDir():
				if strings.EqualFold(filepath.Ext(name), ".momento") {
					continue
				}
				if err := walk(path, append(slices.Clone(folders), name)); err != nil {
					return err
				}
			case !mode.IsRegular():
			case Supported(path):
				if err := add(path, folders); err != nil {
					return err
				}
			default:
				unsupported(path)
			}
		}
		return nil
	}
	for _, root := range paths {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		info, err := os.Stat(root)
		if err != nil {
			return out, err
		}
		if info.IsDir() {
			if err := walk(root, nil); err != nil {
				return out, err
			}
			continue
		}
		if !info.Mode().IsRegular() || strings.HasPrefix(filepath.Base(root), ".") {
			continue
		}
		if !Supported(root) {
			unsupported(root)
			absolute, err := filepath.Abs(root)
			if err != nil {
				return out, err
			}
			out.direct = append(out.direct, absolute)
			continue
		}
		if err := add(root, nil); err != nil {
			return out, err
		}
	}
	sort.Slice(out.candidates, func(i, j int) bool { return out.candidates[i].path < out.candidates[j].path })
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

// ioWorkers 是复制和算哈希这类读写文件任务的并发数。
func ioWorkers() int { return min(max(runtime.NumCPU(), 2), 6) }

// parallel 用最多 workers 个协程跑 count 个任务。completed 在调用方协程里逐个串行调用。
func parallel(ctx context.Context, count, workers int, work func(context.Context, int) error, completed ...func(int)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	workers = min(count, workers)
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
	if c.motion != nil {
		return stageMotion(ctx, c.motion, stage, index)
	}
	return nil
}

// stageMotion 把实况照片的视频复制进暂存并算哈希，放进图库时用来核对已有文件。
func stageMotion(ctx context.Context, m *motionSource, stage string, index int) error {
	in, err := os.Open(m.path)
	if err != nil {
		return sourceError{err}
	}
	defer in.Close()
	m.staged = filepath.Join(stage, fmt.Sprintf("source-%d.motion.mov", index))
	out, err := os.OpenFile(m.staged, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(out, h), contextReader{ctx: ctx, r: sourceReader{in}})
	if err = errors.Join(err, out.Close()); err != nil {
		return err
	}
	m.hash = hex.EncodeToString(h.Sum(nil))
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

// Import 整批导入。单张图片读不了、解不开或太大时只跳过这一张，记进 result.Errors，源文件路径同序记进 result.Failed；
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
	found, err := scan(ctx, request.Paths)
	if err != nil {
		return result, err
	}
	candidates := pairLivePhotos(found.candidates)
	total := len(candidates)
	progress.send(model.ImportProgress{Stage: "scanning", Total: total}, true)
	if total == 0 {
		progress.send(model.ImportProgress{Stage: "finishing"}, true)
		// 有文件但一个都不支持：报出来，不能显示「已导入 0 个」让用户以为没反应。
		if found.unsupported > 0 {
			return result, i18n.NewError("error_import_unsupported", found.unsupported, found.example)
		}
		return result, nil
	}
	// 和图片一起直接拖进来的不支持文件，逐个记成没能导入，结束时和坏图一起列出来。
	defer func() {
		if err != nil {
			return
		}
		for _, path := range found.direct {
			slog.Warn("导入时跳过不支持的文件", "file", filepath.Base(path))
			result.Errors = append(result.Errors, i18n.NewError("error_import_unsupported_file", filepath.Base(path)))
			result.Failed = append(result.Failed, path)
		}
	}()
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
	if err = parallel(ctx, total, ioWorkers(), func(ctx context.Context, i int) error {
		if pairErr := candidates[i].pairErr; pairErr != nil {
			fail(i, pairErr)
			return nil
		}
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
	// 只查这一批的哈希在不在库里，不读整个库：扩展一张张导入时，十万张的库每张都全表读一遍太慢。
	existing, err := s.existingAssets(ctx, hashes)
	if err != nil {
		return result, err
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
			// 视频不读关键词（XMP 关键词只在图片里读）。
			if fileErrs[groups[hash][0]] != nil || isVideo(c.extension) {
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
	// 解码按内存预算排队，小图可以并行到 CPU 数，大图同时只解几张。
	if err = parallel(ctx, len(toPrepare), decodeWorkers(), func(ctx context.Context, j int) error {
		hash := toPrepare[j]
		c := candidates[groups[hash][0]]
		prepare := s.prepareImage
		if isVideo(c.extension) {
			prepare = s.prepareVideo
		}
		img, err := prepare(ctx, c.staged)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			prepareErrs[j] = err
			return nil
		}
		base := filepath.Join(stage, c.hash)
		if repair[hash] {
			// 补回的缩略图要带上图库里已有的旋转。
			rotated, err := RotateImage(img.thumbnail, existing[hash].rotation)
			if err != nil {
				return err
			}
			thumb, err := WriteThumbnail(base, rotated)
			if err != nil {
				return err
			}
			ready[j] = prepared{original: c.staged, thumbnail: thumb}
			return nil
		}
		metadata := img.metadata
		if !c.createdAt.IsZero() {
			metadata["fileCreatedAt"] = c.createdAt.UTC().Format(time.RFC3339Nano)
		} else {
			delete(metadata, "fileCreatedAt")
		}
		thumb, err := WriteThumbnail(base, img.thumbnail)
		if err != nil {
			return err
		}
		name := filepath.Base(c.path)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		now := time.Now().UTC()
		kind, durationMS, motionPath := img.kind, img.durationMS, ""
		if kind == "" {
			kind = model.KindImage
		}
		if c.motion != nil {
			kind, durationMS, motionPath = model.KindLive, c.motion.durationMS, motionRelative(c.hash, "mov")
		}
		// 通用封面只是纯色底，取不出有意义的颜色，不参与按颜色筛选。
		var colors []model.Color
		if metadata["posterUnavailable"] == "" {
			colors = Palette(img.thumbnail)
		}
		// 网页标题可能带换行，名称压成一行。
		if custom := strings.Join(strings.Fields(request.Name), " "); custom != "" {
			name = custom
		}
		ready[j] = prepared{asset: model.Asset{ID: c.hash, Name: name, Note: strings.TrimSpace(request.Note), OriginalFileName: filepath.Base(c.path), Extension: c.extension, MediaType: mediaType(c.extension), Size: c.size, Width: img.width, Height: img.height, Orientation: img.orientation, ColorProfile: img.profile, EXIF: metadata, SourceURL: request.SourceURL, RelativePath: filepath.ToSlash(filepath.Join("assets", c.hash[:2], c.hash+"."+c.extension)), CreatedAt: now, UpdatedAt: now, Colors: colors, Kind: kind, DurationMS: durationMS, MotionPath: motionPath}, original: c.staged, thumbnail: thumb}
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
	folders := importFolders{}
	// 按名字传来的标签：去掉首尾空白和空名字，同名（忽略大小写）只留一个。
	var tagNames []string
	seenTags := map[string]bool{}
	for _, name := range request.TagNames {
		name = strings.TrimSpace(name)
		if key := library.NormalizeTag(name); name != "" && !seenTags[key] {
			seenTags[key] = true
			tagNames = append(tagNames, name)
		}
	}
	// 配对视频单独出的问题（图库里这张图已配了别的视频），图片本身照常算导入或重复。
	motionFailed := map[int]error{}
	repaired := map[string]bool{}
	resultIDs := map[string]bool{}
	for i, c := range candidates {
		if fileErrs[i] != nil {
			continue
		}
		var trashed bool
		var source, relative, motionPath string
		lookupErr := tx.QueryRowContext(ctx, "SELECT trashed,source_url,relative_path,motion_path FROM assets WHERE id=?", c.hash).Scan(&trashed, &source, &relative, &motionPath)
		switch {
		case errors.Is(lookupErr, sql.ErrNoRows):
			p, ok := byHash[c.hash]
			if !ok || repair[c.hash] {
				return result, i18n.NewError("error_import_record_changed", c.hash)
			}
			if err = insertAsset(ctx, tx, p.asset); err != nil {
				return result, err
			}
			type installing struct {
				source, relative, hash string
				original               bool
			}
			files := []installing{{p.original, p.asset.RelativePath, c.hash, true}, {p.thumbnail, thumbnailRelative(c.hash, p.thumbnail), c.hash, false}}
			if m := candidates[i].motion; p.asset.MotionPath != "" && m != nil {
				files = append(files, installing{m.staged, p.asset.MotionPath, m.hash, true})
			}
			for _, file := range files {
				dst, placed, installErr := s.installFile(ctx, file.source, file.relative, file.hash, file.original)
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
				}{{p.original, relative, true}, {p.thumbnail, thumbnailRelative(c.hash, p.thumbnail), false}} {
					if _, _, err = s.installFile(ctx, file.source, file.relative, c.hash, file.original); err != nil {
						return result, err
					}
				}
				repaired[c.hash] = true
				slog.Info("已补回图库缺失的原图和缩略图", "id", c.hash, "path", relative)
			}
			repairedMotion := false
			if m := c.motion; m != nil {
				if motionPath == "" {
					// 图库里已有这张图但没有视频（比如以前单独导入过）：补存视频，升级成实况照片，源视频才能放心移走。
					rel := motionRelative(c.hash, "mov")
					dst, placed, installErr := s.installFile(ctx, m.staged, rel, m.hash, true)
					if placed {
						installed = append(installed, dst)
					}
					if installErr != nil {
						return result, installErr
					}
					if _, err = tx.ExecContext(ctx, "UPDATE assets SET kind=?,duration_ms=?,motion_path=?,updated_at=? WHERE id=?", model.KindLive, m.durationMS, rel, nowText(), c.hash); err != nil {
						return result, err
					}
					slog.Info("已有的图片补上实况视频", "id", c.hash)
				} else if stored, pathErr := s.assetPath(motionPath); errors.Is(pathErr, os.ErrNotExist) {
					dst, placed, installErr := s.installFile(ctx, m.staged, motionPath, m.hash, true)
					if placed {
						installed = append(installed, dst)
					}
					if installErr != nil {
						return result, installErr
					}
					if _, err = tx.ExecContext(ctx, "UPDATE assets SET duration_ms=?,updated_at=? WHERE id=?", m.durationMS, nowText(), c.hash); err != nil {
						return result, err
					}
					repairedMotion = true
					slog.Warn("图库里缺少实况视频，用这次导入的视频补回", "id", c.hash, "path", motionPath)
				} else if pathErr != nil {
					return result, pathErr
				} else {
					// 已经有视频：内容一样才算重复；不一样的视频不能当成已入库，留着源文件报出来。
					same, hashErr := fileHashEquals(ctx, stored, m.hash)
					if hashErr != nil && !errors.Is(hashErr, os.ErrNotExist) {
						return result, hashErr
					}
					if !same {
						slog.Warn("这张实况照片在图库里已配了另一段视频，这段视频没有导入", "id", c.hash, "file", filepath.Base(m.path))
						motionFailed[i] = i18n.NewError("error_live_motion_conflict", filepath.Base(m.path))
					}
				}
			}
			if trashed {
				result.Imported++
				result.Restored++
			} else if repairedMotion {
				result.Imported++
				skipped--
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
		if c.motion != nil && motionFailed[i] == nil {
			result.Motion = append(result.Motion, c.motion.path)
			result.MotionOwners = append(result.MotionOwners, model.MotionOwner{SourcePath: c.motion.path, AssetID: c.hash})
		}
		for _, tagID := range request.TagIDs {
			if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO asset_tags(asset_id,tag_id) VALUES(?,?)", c.hash, tagID); err != nil {
				return result, err
			}
		}
		if err = attachKeywords(ctx, tx, c.hash, keywordsByHash[c.hash]); err != nil {
			return result, err
		}
		if err = attachKeywords(ctx, tx, c.hash, tagNames); err != nil {
			return result, err
		}
		parent := request.FolderID
		for _, name := range c.folders {
			parent, err = folders.ensureFolder(ctx, tx, name, parent)
			if err != nil {
				return result, err
			}
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
	// Failed 和 Errors 按源文件一一对应，重复内容一起失败时每个文件各占一条；实况照片没导入时，配对的视频也算一条。
	failedCandidates := 0
	for i, fileErr := range fileErrs {
		if fileErr == nil {
			continue
		}
		failedCandidates++
		result.Errors = append(result.Errors, fileErr)
		result.Failed = append(result.Failed, candidates[i].path)
		if m := candidates[i].motion; m != nil {
			var cause *i18n.LocalizedError
			motionErr := i18n.NewError("error_import_read", filepath.Base(m.path))
			if errors.As(fileErr, &cause) && cause.Cause != nil {
				motionErr = i18n.WrapError(cause.Cause, "error_import_read", filepath.Base(m.path))
			}
			result.Errors = append(result.Errors, motionErr)
			result.Failed = append(result.Failed, m.path)
		}
	}
	for _, i := range slices.Sorted(maps.Keys(motionFailed)) {
		result.Errors = append(result.Errors, motionFailed[i])
		result.Failed = append(result.Failed, candidates[i].motion.path)
	}
	progress.send(model.ImportProgress{Stage: "finishing", Total: total, Processed: total, Imported: total - result.Skipped - failedCandidates, Skipped: result.Skipped}, true)
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
		if !original {
			// 换了格式的缩略图（比如旧的 .png 换成 .jpg）不留旧的那份。
			err = removeOtherThumbnails(strings.TrimSuffix(dst, filepath.Ext(dst)), filepath.Ext(dst))
		}
		return dst, true, err
	case statErr != nil:
		return dst, false, statErr
	case !info.Mode().IsRegular():
		return dst, false, i18n.NewError("error_import_unregistered_file", relative)
	case !original:
		if err = os.Rename(staged, dst); err != nil {
			return dst, false, err
		}
		return dst, true, removeOtherThumbnails(strings.TrimSuffix(dst, filepath.Ext(dst)), filepath.Ext(dst))
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

// thumbnailRelative 是暂存缩略图放进图库后的相对路径，沿用暂存文件的后缀。
func thumbnailRelative(id, staged string) string {
	return filepath.Join("thumbnails", id+filepath.Ext(staged))
}

func mediaType(ext string) string {
	if isVideo(ext) {
		return videoMediaType(ext)
	}
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "tif", "tiff":
		return "image/tiff"
	case "heic", "heif":
		return "image/heic"
	case "psd":
		return "image/vnd.adobe.photoshop"
	case "ico":
		return "image/x-icon"
	case "dng":
		return "image/x-adobe-dng"
	case "hdr":
		return "image/vnd.radiance"
	}
	if media.IsRawExtension(ext) || media.IsSystemImageExtension(ext) {
		return "image/x-" + ext
	}
	return "image/" + ext
}

func insertAsset(ctx context.Context, tx *sql.Tx, a model.Asset) error {
	metadata, err := json.Marshal(a.EXIF)
	if err != nil {
		return err
	}
	kind := a.Kind
	if kind == "" {
		kind = model.KindImage
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO assets(id,name,original_file_name,extension,media_type,size,width,height,orientation,color_profile,exif_json,note,source_url,relative_path,favorite,trashed,created_at,updated_at,kind,duration_ms,motion_path) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, a.ID, a.Name, a.OriginalFileName, a.Extension, a.MediaType, a.Size, a.Width, a.Height, a.Orientation, a.ColorProfile, string(metadata), a.Note, a.SourceURL, a.RelativePath, a.Favorite, a.Trashed, a.CreatedAt.Format(time.RFC3339Nano), a.UpdatedAt.Format(time.RFC3339Nano), kind, a.DurationMS, a.MotionPath)
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

type importFolders map[string]*importFolderChildren

type importFolderChildren struct {
	byName   map[string]string
	nextSort int
}

const importFolderChildrenQuery = "SELECT id,name,sort_index FROM folders WHERE parent_id IS ? ORDER BY sort_index,id"

func (folders importFolders) ensureFolder(ctx context.Context, tx *sql.Tx, name, parent string) (string, error) {
	var parentValue any
	if parent != "" {
		parentValue = parent
	}
	children := folders[parent]
	if children == nil {
		rows, err := tx.QueryContext(ctx, importFolderChildrenQuery, parentValue)
		if err != nil {
			return "", err
		}
		children = &importFolderChildren{byName: map[string]string{}}
		for rows.Next() {
			var id, name string
			var sortIndex int
			if err = rows.Scan(&id, &name, &sortIndex); err != nil {
				return "", errors.Join(err, rows.Close())
			}
			if _, exists := children.byName[name]; !exists {
				children.byName[name] = id
			}
			children.nextSort = sortIndex + 1
		}
		if err = errors.Join(rows.Err(), rows.Close()); err != nil {
			return "", err
		}
		folders[parent] = children
	}
	if id, exists := children.byName[name]; exists {
		return id, nil
	}
	id := rand.Text()
	if _, err := tx.ExecContext(ctx, "INSERT INTO folders(id,name,parent_id,sort_index) VALUES(?,?,?,?)", id, name, parentValue, children.nextSort); err != nil {
		return "", err
	}
	children.byName[name] = id
	children.nextSort++
	return id, nil
}

// existingAssets 按素材 ID（内容哈希）查库里已有的素材，分批查，免得超过 SQLite 的参数个数上限。
func (s *Service) existingAssets(ctx context.Context, ids []string) (map[string]existingAsset, error) {
	existing := make(map[string]existingAsset, len(ids))
	const chunk = 500
	for start := 0; start < len(ids); start += chunk {
		part := ids[start:min(start+chunk, len(ids))]
		args := make([]any, len(part))
		for i, id := range part {
			args[i] = id
		}
		rows, err := s.db.QueryContext(ctx, "SELECT id,trashed,relative_path,rotation FROM assets WHERE id IN (?"+strings.Repeat(",?", len(part)-1)+")", args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id string
			var a existingAsset
			if err := rows.Scan(&id, &a.trashed, &a.relative, &a.rotation); err != nil {
				return nil, errors.Join(err, rows.Close())
			}
			existing[id] = a
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return nil, err
		}
	}
	return existing, nil
}
