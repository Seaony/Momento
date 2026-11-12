package ui

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// 文件路径和进程内 Drag 值可以同时挂在一张卡片上。
func AttachFileDrag(element ui.Element, root string, selected func() []model.Asset, report func(error)) {
	var session *fileDragSession
	preparationFailed := false
	fail := func(err error) {
		slog.Error("拖出素材失败", "error", err)
		report(err)
	}
	element.DragDataFrom(func() transfer.Data {
		var err error
		session, err = prepareFileDrag(root, selected(), os.Link)
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
		slog.Info("开始拖出素材", "count", len(session.paths))
		return data
	}, transfer.DragOptions{
		Operations: transfer.Copy,
		Done: func(result transfer.Result) {
			var err error
			if session != nil {
				err = session.cleanup()
				session = nil
			}
			if !preparationFailed {
				err = errors.Join(result.Err, err)
			}
			if err != nil {
				fail(err)
			}
		},
	})
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
