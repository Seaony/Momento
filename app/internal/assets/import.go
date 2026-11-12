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
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	c.createdAt, _ = sourceCreatedAt(info)
	c.staged = filepath.Join(stage, fmt.Sprintf("source-%d.%s", index, c.extension))
	out, err := os.OpenFile(c.staged, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	h := sha256.New()
	c.size, err = io.Copy(io.MultiWriter(out, h), contextReader{ctx: ctx, r: in})
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
			slog.Info("导入完成", "imported", result.Imported, "restored", result.Restored, "skipped", result.Skipped)
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
	hashed := 0
	if err = parallel(ctx, total, func(ctx context.Context, i int) error { return stageHash(ctx, &candidates[i], stage, i) }, func(i int) {
		hashed++
		progress.send(model.ImportProgress{Stage: "scanning", Total: total, Processed: hashed, CurrentFile: filepath.Base(candidates[i].path)}, false)
	}); err != nil {
		return result, err
	}
	existing := map[string]bool{}
	rows, err := s.db.QueryContext(ctx, "SELECT id,trashed FROM assets")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var id string
		var trashed bool
		if err = rows.Scan(&id, &trashed); err != nil {
			rows.Close()
			return result, err
		}
		existing[id] = trashed
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return result, err
	}
	groups := map[string][]int{}
	var fresh []int
	for i, c := range candidates {
		if len(groups[c.hash]) == 0 {
			if _, exists := existing[c.hash]; !exists {
				fresh = append(fresh, i)
			}
		}
		groups[c.hash] = append(groups[c.hash], i)
	}
	keywordsByHash := map[string][]string{}
	if request.ImportKeywords {
		for hash, indices := range groups {
			if err = ctx.Err(); err != nil {
				return result, err
			}
			c := candidates[indices[0]]
			keywordsByHash[hash], err = ReadKeywords(c.staged)
			if err != nil {
				return result, i18n.WrapError(err, "error_import_read", filepath.Base(c.path))
			}
		}
	}
	processed, skipped := 0, 0
	for hash, indices := range groups {
		if trashed, exists := existing[hash]; exists {
			processed += len(indices)
			skipped += len(indices)
			if trashed {
				skipped--
			}
		}
	}
	ready := make([]prepared, len(fresh))
	// 进度里的导入数统计已准备好的非重复素材，最终结果仍以整批提交为准。
	progress.send(model.ImportProgress{Stage: "importing", Total: total, Processed: processed, Imported: processed - skipped, Skipped: skipped}, true)
	if err = parallel(ctx, len(fresh), func(ctx context.Context, j int) error {
		c := candidates[fresh[j]]
		img, metadata, orientation, profile, err := s.prepareImage(c.staged)
		if err != nil {
			return i18n.WrapError(err, "error_import_read", filepath.Base(c.path))
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if !c.createdAt.IsZero() {
			metadata["fileCreatedAt"] = c.createdAt.UTC().Format(time.RFC3339Nano)
		} else {
			delete(metadata, "fileCreatedAt")
		}
		thumb := filepath.Join(stage, c.hash+".png")
		if err = WriteThumbnail(thumb, img); err != nil {
			return err
		}
		name := filepath.Base(c.path)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		now := time.Now().UTC()
		ready[j] = prepared{asset: model.Asset{ID: c.hash, Name: name, OriginalFileName: filepath.Base(c.path), Extension: c.extension, MediaType: mediaType(c.extension), Size: c.size, Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), Orientation: orientation, ColorProfile: profile, EXIF: metadata, SourceURL: request.SourceURL, RelativePath: filepath.ToSlash(filepath.Join("assets", c.hash[:2], c.hash+"."+c.extension)), CreatedAt: now, UpdatedAt: now, Colors: Palette(img)}, original: c.staged, thumbnail: thumb}
		return nil
	}, func(j int) {
		c := candidates[fresh[j]]
		count := len(groups[c.hash])
		processed += count
		skipped += count - 1
		progress.send(model.ImportProgress{Stage: "importing", Total: total, Processed: processed, Imported: processed - skipped, Skipped: skipped, CurrentFile: filepath.Base(c.path)}, false)
	}); err != nil {
		return result, err
	}
	progress.send(model.ImportProgress{Stage: "importing", Total: total, Processed: processed, Imported: processed - skipped, Skipped: skipped}, true)
	progress.send(model.ImportProgress{Stage: "finishing", Total: total, Processed: processed, Imported: processed - skipped, Skipped: skipped}, true)
	byHash := make(map[string]prepared, len(ready))
	for _, p := range ready {
		byHash[p.asset.ID] = p
	}
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
	verified := map[string]bool{}
	resultIDs := map[string]bool{}
	for _, c := range candidates {
		var trashed bool
		var source, relative string
		lookupErr := tx.QueryRowContext(ctx, "SELECT trashed,source_url,relative_path FROM assets WHERE id=?", c.hash).Scan(&trashed, &source, &relative)
		switch {
		case errors.Is(lookupErr, sql.ErrNoRows):
			p, ok := byHash[c.hash]
			if !ok {
				return result, i18n.NewError("error_import_record_changed", c.hash)
			}
			if err = insertAsset(ctx, tx, p.asset); err != nil {
				return result, err
			}
			for _, file := range []struct{ source, relative string }{{p.original, p.asset.RelativePath}, {p.thumbnail, filepath.Join("thumbnails", c.hash+".png")}} {
				dst, destinationErr := s.storageDestination(file.relative)
				if destinationErr != nil {
					return result, destinationErr
				}
				if _, statErr := os.Lstat(dst); statErr == nil {
					return result, i18n.NewError("error_import_unregistered_file", file.relative)
				} else if !os.IsNotExist(statErr) {
					return result, statErr
				}
				if err = publishFile(file.source, dst); err != nil {
					return result, err
				}
				installed = append(installed, dst)
			}
			result.Imported++
			verified[c.hash] = true
		case lookupErr != nil:
			return result, lookupErr
		default:
			if !verified[c.hash] {
				if err = s.verifyDuplicate(ctx, relative, c.hash); err != nil {
					return result, i18n.WrapError(err, "error_import_read", filepath.Base(c.path))
				}
				verified[c.hash] = true
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
		progress.send(model.ImportProgress{Stage: "finishing", Total: total, Processed: processed, Imported: processed - skipped, Skipped: skipped, CurrentFile: filepath.Base(c.path)}, false)
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	committed = true
	progress.send(model.ImportProgress{Stage: "finishing", Total: total, Processed: total, Imported: result.Imported, Skipped: result.Skipped}, true)
	return result, nil
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
