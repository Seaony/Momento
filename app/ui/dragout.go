package ui

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// 文件路径和进程内 Drag 值可以同时挂在一张卡片上。
func AttachFileDrag(element ui.Element, root string, selected func() []model.Asset, report func(error), used func([]string)) {
	var session *fileDragSession
	preparationFailed := false
	fail := func(err error) {
		slog.Error("拖出素材失败", "error", err)
		report(err)
	}
	element.DragDataFrom(func() transfer.Data {
		// 顺手清掉早先拖出留下的临时文件；清不掉只提示，不挡这次拖出。
		if err := sweepDragSessions(root, time.Now(), dragSessionKeep, dragSessionLimit); err != nil {
			fail(err)
		}
		var err error
		items := selected()
		session, err = prepareFileDrag(root, items, os.Link)
		preparationFailed = err != nil
		if err != nil {
			fail(err)
			return transfer.Data{}
		}
		data, err := transfer.FileData(session.paths...)
		if err != nil {
			preparationFailed = true
			fail(errors.Join(err, session.cleanup()))
			session = nil
			return transfer.Data{}
		}
		if used != nil {
			ids := make([]string, 0, len(items))
			for _, item := range items {
				ids = append(ids, item.ID)
			}
			used(ids)
		}
		slog.Info("开始拖出素材", "count", len(session.paths))
		return data
	}, transfer.DragOptions{
		Operations: transfer.Copy,
		Done: func(result transfer.Result) {
			err := finishFileDrag(session, result)
			session = nil
			if !preparationFailed {
				err = errors.Join(result.Err, err)
			}
			if err != nil {
				fail(err)
			}
		},
	})
}

const (
	dragSessionKeep  = 10 * time.Minute // 放下后临时文件至少留这么久，给事后才读文件的应用
	dragSessionLimit = 20               // 最多留这么多次拖出的临时目录，超出时先删最旧的
)

// ClearDragSessions 删掉图库里所有拖出临时文件，图库关闭或应用退出时调用。
func ClearDragSessions(root string) error {
	return sweepDragSessions(root, time.Now(), 0, 0)
}

// 删掉 .drag-out 里超过 keep 的拖出临时目录；剩下的超过 limit-1 个时再删最旧的，给马上要建的新目录留位置。limit 为 0 时不留。
func sweepDragSessions(root string, now time.Time, keep time.Duration, limit int) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	parent := filepath.Join(root, ".drag-out")
	info, err := os.Lstat(parent)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// 不是普通目录（比如被换成指向别处的链接）时不碰，拖出准备会报错。
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	type session struct {
		path     string
		modified time.Time
	}
	var kept []session
	var errs []error
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "session-") {
			continue
		}
		info, err := entry.Info()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		path := filepath.Join(parent, entry.Name())
		if now.Sub(info.ModTime()) >= keep {
			errs = append(errs, os.RemoveAll(path))
			continue
		}
		kept = append(kept, session{path, info.ModTime()})
	}
	if len(kept) > max(0, limit-1) {
		slices.SortFunc(kept, func(x, y session) int { return x.modified.Compare(y.modified) })
		for _, old := range kept[:len(kept)-max(0, limit-1)] {
			errs = append(errs, os.RemoveAll(old.path))
		}
	}
	return errors.Join(errs...)
}

// 没放下就没人读，马上删；放下了的话浏览器、Slack、访达常常过一会儿才读文件，留到之后的拖出或图库关闭时再清。
func finishFileDrag(session *fileDragSession, result transfer.Result) error {
	if session == nil || !result.Canceled && result.Operation != transfer.None {
		return nil
	}
	return session.cleanup()
}

type fileDragSession struct {
	directory string
	paths     []string
}

func (s *fileDragSession) cleanup() error { return os.RemoveAll(s.directory) }

func prepareFileDrag(root string, items []model.Asset, link func(string, string) error) (session *fileDragSession, err error) {
	if len(items) == 0 {
		return nil, i18n.NewError("error_drag_empty")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	parent := filepath.Join(root, ".drag-out")
	if err := os.Mkdir(parent, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	info, err := os.Lstat(parent)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, i18n.NewError("error_drag_directory")
	}
	directory, err := os.MkdirTemp(parent, "session-")
	if err != nil {
		return nil, err
	}
	session = &fileDragSession{directory: directory}
	defer func() {
		if err != nil {
			err = errors.Join(err, session.cleanup())
			session = nil
		}
	}()
	used := make(map[string]bool, len(items))
	for _, asset := range items {
		if !filepath.IsLocal(filepath.FromSlash(asset.RelativePath)) {
			return session, i18n.NewError("error_asset_path_escape", asset.Name)
		}
		source, e := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(asset.RelativePath)))
		if e != nil {
			return session, i18n.WrapError(e, "error_import_read", asset.Name)
		}
		relative, e := filepath.Rel(root, source)
		if e != nil || !filepath.IsLocal(relative) {
			return session, i18n.NewError("error_asset_path_escape", asset.Name)
		}
		info, e := os.Stat(source)
		if e != nil {
			return session, e
		}
		if !info.Mode().IsRegular() {
			return session, i18n.NewError("error_asset_file", asset.Name)
		}
		name := dragFileName(asset)
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		for n := 2; used[strings.ToLower(name)]; n++ {
			name = fmt.Sprintf("%s %d%s", stem, n, ext)
		}
		used[strings.ToLower(name)] = true
		destination := filepath.Join(directory, name)
		if e := link(source, destination); e != nil {
			slog.Debug("拖出硬链接不可用，改为复制", "error", e)
			if e := copyDragFile(source, destination); e != nil {
				return session, i18n.WrapError(e, "error_drag_prepare", name)
			}
		}
		session.paths = append(session.paths, destination)
	}
	return session, nil
}

func dragFileName(asset model.Asset) string {
	name := asset.OriginalFileName
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSpace(asset.Name)
		if name == "" {
			name = "image"
		}
		if ext := strings.TrimPrefix(asset.Extension, "."); ext != "" {
			name += "." + ext
		}
	}
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, name)
	if name == "." || name == ".." {
		name = "image"
	}
	return name
}

func copyDragFile(source, destination string) (err error) {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, in.Close()) }()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	_, err = io.Copy(out, in)
	return err
}
