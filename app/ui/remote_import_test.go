package ui

import (
	"errors"
	"testing"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestRemoteBatchFeedbackAfterAllItems(t *testing.T) {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.Notify = false
	item := func(i int) browser.Request { return browser.Request{BatchID: "b1", Index: i, Total: 3} }
	// 乱序到达：编号最大的先做完也不能提前汇总。
	a.remoteFeedback(item(2), model.ImportResult{Imported: 1, AssetIDs: []string{"a"}}, nil)
	a.remoteFeedback(item(0), model.ImportResult{}, i18n.NewError("error_download_status", 403))
	if a.Notice != "" || a.Error != "" {
		t.Fatalf("没全部做完就提示了：%q %q", a.Notice, a.Error)
	}
	a.remoteFeedback(item(1), model.ImportResult{Skipped: 1, AssetIDs: []string{"c"}}, nil)
	if a.Notice != a.T("import_complete_failed", 1, 1, 1) {
		t.Fatalf("汇总提示是 %q", a.Notice)
	}
	if len(a.remoteBatches) != 0 {
		t.Fatal("这一批结束后记录没清掉")
	}
}

func TestRemoteBatchCanceledStaysQuiet(t *testing.T) {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.Notify = false
	for i := range 2 {
		a.remoteFeedback(browser.Request{BatchID: "b2", Index: i, Total: 2}, model.ImportResult{}, i18n.NewError("import_canceled"))
	}
	if a.Notice != "" || a.Error != "" {
		t.Fatalf("取消整批后还提示了：%q %q", a.Notice, a.Error)
	}
	// 单张：扩展说不要提示就不提示，失败照样报。
	quiet := false
	a.remoteFeedback(browser.Request{PlayFeedback: &quiet}, model.ImportResult{Imported: 1}, nil)
	if a.Notice != "" {
		t.Fatalf("不要提示时提示了 %q", a.Notice)
	}
	a.remoteFeedback(browser.Request{}, model.ImportResult{}, errors.New("boom"))
	if a.Error == "" {
		t.Fatal("单张失败没报错")
	}
}
