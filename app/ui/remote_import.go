package ui

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
)

// remoteBusyPoll 是本机正忙（别的导入在写库）时，扩展的导入隔多久再看一次能不能开始。
const remoteBusyPoll = 200 * time.Millisecond

// remoteBatchExpiry 是一批导入多久没有新的一张就当作扔了（扩展中途被关掉），清掉记录。
const remoteBatchExpiry = 10 * time.Minute

// remoteBatch 是浏览器扩展同一批导入（比如一整篇笔记）的进度：放哪儿只问一次，结果攒到最后一张一起通知。
type remoteBatch struct {
	choice   importChoice
	chosen   bool
	canceled bool
	// gen 是选位置时的图库：中途换了图库，旧的文件夹和标签不能再用，要重新问。
	gen     uint64
	result  model.ImportResult
	done    int
	updated time.Time
}

// ImportRemote 处理浏览器扩展和深度链接的一次导入，等真实结果再返回，不能排上队就说成功。
// 本机正在导入时排队等着，不再回「忙」。
func (a *App) ImportRemote(ctx context.Context, r browser.Request) error {
	choice, err := a.remoteChoice(ctx, r)
	if err != nil {
		a.Window.Update(func() { a.remoteFeedback(r, model.ImportResult{}, err) })
		return err
	}
	var service *assets.Service
	var lib *library.Library
	var gen uint64
	var request model.ImportRequest
	for {
		readyErr, waiting := error(context.Canceled), false
		mygo.RunOnMain(func() {
			switch {
			case a.Library == nil:
				readyErr = i18n.NewError("library_required")
			case a.Busy:
				readyErr, waiting = nil, true
			default:
				readyErr = nil
				a.Busy = true
				service, lib, gen = a.Assets, a.Library, a.generation
				request = model.ImportRequest{FolderID: choice.FolderID, TagIDs: choice.TagIDs, ImportKeywords: a.Preferences.ImportKeywords}
			}
		})
		if readyErr != nil {
			a.Window.Update(func() { a.remoteFeedback(r, model.ImportResult{}, readyErr) })
			return readyErr
		}
		if !waiting {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(remoteBusyPoll):
		}
	}
	return a.performRemote(ctx, service, lib, gen, r, request)
}

// remoteChoice 在开着「保存时选位置」时问放到哪儿；同一批只问第一张，后面沿用（取消了就整批取消）。
func (a *App) remoteChoice(ctx context.Context, r browser.Request) (importChoice, error) {
	picker, reused, canceled := false, importChoice{}, false
	found := false
	mygo.RunOnMain(func() {
		picker = a.Preferences.ExtensionPicker && !a.IsLocked()
		if b := a.remoteBatch(r.BatchID); picker && b != nil && (b.chosen || b.canceled) && b.gen == a.generation {
			found, reused, canceled = true, b.choice, b.canceled
		}
	})
	switch {
	case !picker:
		return importChoice{}, nil
	case found && canceled:
		return importChoice{}, i18n.NewError("import_canceled")
	case found:
		return reused, nil
	}
	var gen uint64
	mygo.RunOnMain(func() { gen = a.generation })
	choice, err := a.remoteImportChoice(ctx)
	mygo.RunOnMain(func() {
		if b := a.remoteBatch(r.BatchID); b != nil {
			b.choice, b.chosen, b.gen = choice, err == nil, gen
			b.canceled = err != nil && hasKey(err, "import_canceled")
		}
	})
	return choice, err
}

func (a *App) performRemote(ctx context.Context, service *assets.Service, lib *library.Library, gen uint64, r browser.Request, request model.ImportRequest) error {
	a.workMu.Lock()
	defer a.workMu.Unlock()
	request.Name, request.Note, request.TagNames = r.Title, r.Note, r.Tags
	var result model.ImportResult
	var err error
	if r.File != "" {
		result, err = service.ImportUpload(ctx, r.File, r.FileName, r.PageURL, request)
	} else {
		result, err = service.ImportDownload(ctx, assets.Download{URL: r.URL, PageURL: r.PageURL, Referer: r.Referer, Video: r.Kind == "video"}, request)
	}
	var snapshot model.Snapshot
	if err == nil {
		snapshot, err = lib.Snapshot(ctx)
	}
	a.Window.Update(func() {
		if gen != a.generation {
			return
		}
		a.Busy = false
		if err == nil {
			a.applySnapshot(snapshot)
		}
		a.remoteFeedback(r, result, err)
		a.drainPending()
	})
	return err
}

// remoteBatch 取一批的记录，没有就新建；没有批号返回 nil。顺手清掉早就没动静的批次。只在主线程调。
func (a *App) remoteBatch(id string) *remoteBatch {
	if id == "" {
		return nil
	}
	now := time.Now()
	if a.remoteBatches == nil {
		a.remoteBatches = map[string]*remoteBatch{}
	}
	b := a.remoteBatches[id]
	if b == nil {
		for key, old := range a.remoteBatches {
			if now.Sub(old.updated) > remoteBatchExpiry {
				delete(a.remoteBatches, key)
			}
		}
		b = &remoteBatch{}
		a.remoteBatches[id] = b
	}
	b.updated = now
	return b
}

// remoteFeedback 告诉用户导入结果：单张按扩展要求通知；同一批攒到全部做完一起通知一次，失败的一起列出来。只在主线程调。
func (a *App) remoteFeedback(r browser.Request, result model.ImportResult, err error) {
	canceled := hasKey(err, "import_canceled") || errors.Is(err, context.Canceled)
	b := a.remoteBatch(r.BatchID)
	if b == nil {
		switch {
		case canceled:
		case err != nil:
			a.ShowError(err)
		case r.PlayFeedback == nil || *r.PlayFeedback:
			a.importFeedback("extension", result)
		}
		return
	}
	b.result.Imported += result.Imported
	b.result.Skipped += result.Skipped
	b.result.Restored += result.Restored
	b.result.AssetIDs = append(b.result.AssetIDs, result.AssetIDs...)
	if err != nil && !canceled {
		b.result.Errors = append(b.result.Errors, err)
	}
	// 按做完的张数算，不按编号：请求可能乱序到达。
	b.done++
	if r.Total > 0 && b.done < r.Total {
		return
	}
	delete(a.remoteBatches, r.BatchID)
	if b.result.Imported+b.result.Skipped+b.result.Restored+len(b.result.Errors) == 0 {
		// 整批都被取消了，不用再通知。
		slog.Info("浏览器扩展这一批导入已取消", "batch", r.BatchID)
		return
	}
	a.importFeedback("extension", b.result)
}

// hasKey 判断错误链上有没有这个本地化错误。
func hasKey(err error, key string) bool {
	var localized *i18n.LocalizedError
	for err != nil {
		if !errors.As(err, &localized) {
			return false
		}
		if localized.Key == key {
			return true
		}
		err = localized.Cause
	}
	return false
}
