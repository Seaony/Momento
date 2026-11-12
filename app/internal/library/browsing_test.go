package library

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func TestBrowsingMetadataPersistsInPortableBackup(t *testing.T) {
	l := newTestLibrary(t)
	asset := addBackupAsset(t, l)
	original, err := os.ReadFile(filepath.Join(l.Root, filepath.FromSlash(asset.RelativePath)))
	if err != nil {
		t.Fatal(err)
	}
	used := time.Date(2026, 11, 12, 15, 16, 17, 123, time.UTC)
	if err := l.MarkUsed(t.Context(), []string{asset.ID}, used); err != nil {
		t.Fatal(err)
	}
	if err := l.SetRotation(t.Context(), []string{asset.ID}, -90); err != nil {
		t.Fatal(err)
	}
	if err := l.RotateAssets(t.Context(), []string{asset.ID}, 180); err != nil {
		t.Fatal(err)
	}
	smart, err := l.CreateSmartFolder(t.Context(), "参考图", "all", []model.SmartRule{{Field: "extension", Operator: "equals", Value: "png"}, {Field: "size", Operator: "gte", Value: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.PinQuickAccess(t.Context(), "smart", smart.ID); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, l)
	got := before.Assets[0]
	if !got.LastUsedAt.Equal(used) || got.Rotation != 90 || len(before.SmartFolders) != 1 || len(before.QuickAccess) != 1 {
		t.Fatal("元数据未保存", before)
	}
	after, err := os.ReadFile(filepath.Join(l.Root, filepath.FromSlash(asset.RelativePath)))
	if err != nil || string(after) != string(original) {
		t.Fatal("旋转改动了原图", err)
	}
	destination := filepath.Join(t.TempDir(), "完整备份.momento")
	if err := l.ExportLibrary(t.Context(), destination); err != nil {
		t.Fatal(err)
	}
	backup, err := Open(t.Context(), destination)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, backup); !reflect.DeepEqual(got, before) {
		t.Fatal("备份丢失浏览数据", got, before)
	}
	backup.Close()
	moved := filepath.Join(t.TempDir(), "移动后的备份.momento")
	if err := os.Rename(destination, moved); err != nil {
		t.Fatal(err)
	}
	backup, err = Open(t.Context(), moved)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if got := snapshot(t, backup); !reflect.DeepEqual(got, before) {
		t.Fatal("移动后浏览数据丢失", got, before)
	}
}

func TestUsageAndRotationBatchesRollback(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	seedAsset(t, l, "b")
	now := time.Now()
	if err := l.MarkUsed(t.Context(), []string{"a", "missing"}, now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if !snapshot(t, l).Assets[0].LastUsedAt.IsZero() {
		t.Fatal("失败批量使用仍更新前项")
	}
	if err := l.SetRotation(t.Context(), []string{"a", "missing"}, 90); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if snapshot(t, l).Assets[0].Rotation != 0 {
		t.Fatal("失败批量旋转未回滚")
	}
	if err := l.SetRotation(t.Context(), []string{"a"}, 45); err == nil {
		t.Fatal("允许非直角旋转")
	}
	if err := l.RotateAssets(t.Context(), []string{"a"}, 45); err == nil {
		t.Fatal("允许非直角增量")
	}
	if err := l.RotateAssets(t.Context(), []string{"a"}, -90); err != nil {
		t.Fatal(err)
	}
	if err := l.RotateAssets(t.Context(), []string{"a", "missing"}, 90); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if snapshot(t, l).Assets[0].Rotation != 270 {
		t.Fatal("旋转增量失败未回滚")
	}
	if err := l.MarkUsed(t.Context(), []string{"a"}, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if snapshot(t, l).Assets[0].LastUsedAt.IsZero() {
		t.Fatal("空时间没有采用当前时间")
	}
}

func TestSmartFolderEditsValidateAndDeletePins(t *testing.T) {
	l := newTestLibrary(t)
	rules := []model.SmartRule{{Field: "note", Operator: "contains", Value: "设计"}}
	smart, err := l.CreateSmartFolder(t.Context(), "  参考  ", "all", rules)
	if err != nil {
		t.Fatal(err)
	}
	if smart.Name != "参考" {
		t.Fatal(smart)
	}
	pin, err := l.PinQuickAccess(t.Context(), "smart", smart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.UpdateSmartFolder(t.Context(), smart.ID, "新参考", "any", []model.SmartRule{{Field: "width", Operator: "gte", Value: "1920"}}); err != nil {
		t.Fatal(err)
	}
	if err := l.RenameSmartFolder(t.Context(), smart.ID, "宽屏"); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, l)
	if err := l.UpdateSmartFolder(t.Context(), smart.ID, "坏规则", "all", []model.SmartRule{{Field: "size", Operator: "gte", Value: "bad"}}); err == nil {
		t.Fatal("无效条件保存成功")
	}
	if _, err := l.CreateSmartFolder(t.Context(), "空规则", "all", nil); err == nil {
		t.Fatal("空规则保存成功")
	}
	if _, err := l.CreateSmartFolder(t.Context(), "", "all", rules); err == nil {
		t.Fatal("空名称保存成功")
	}
	if got := snapshot(t, l); !reflect.DeepEqual(got, before) {
		t.Fatal("失败的智能编辑改动数据")
	}
	if err := l.DeleteSmartFolder(t.Context(), smart.ID); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, l); len(got.SmartFolders) != 0 || len(got.QuickAccess) != 0 {
		t.Fatal("删除智能文件夹残留固定项", got)
	}
	if err := l.UnpinQuickAccess(t.Context(), pin.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if err := l.RenameSmartFolder(t.Context(), "missing", "不存在"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
}

func TestQuickAccessDedupeMoveAndDeletionLifecycle(t *testing.T) {
	l := newTestLibrary(t)
	parent, err := l.CreateFolder(t.Context(), "父文件夹", "")
	if err != nil {
		t.Fatal(err)
	}
	child, err := l.CreateFolder(t.Context(), "子文件夹", parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	tag, err := l.CreateTag(t.Context(), "标签")
	if err != nil {
		t.Fatal(err)
	}
	first, err := l.PinQuickAccess(t.Context(), "folder", parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := l.PinQuickAccess(t.Context(), "folder", child.ID)
	if err != nil {
		t.Fatal(err)
	}
	third, err := l.PinQuickAccess(t.Context(), "tag", tag.ID)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := l.PinQuickAccess(t.Context(), "tag", tag.ID)
	if err != nil || duplicate.ID != third.ID {
		t.Fatal("重复固定创建了新行", err)
	}
	if _, err := l.PinQuickAccess(t.Context(), "asset", "a"); err == nil {
		t.Fatal("允许固定素材")
	}
	if _, err := l.PinQuickAccess(t.Context(), "folder", "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if err := l.MoveQuickAccess(t.Context(), third.ID, 0); err != nil {
		t.Fatal(err)
	}
	pins := snapshot(t, l).QuickAccess
	if len(pins) != 3 || pins[0].ID != third.ID || pins[1].ID != first.ID || pins[2].ID != second.ID {
		t.Fatal(pins)
	}
	for i, pin := range pins {
		if pin.SortIndex != i {
			t.Fatal(pins)
		}
	}
	if err := l.MoveQuickAccess(t.Context(), "missing", 0); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if err := l.RenameFolder(t.Context(), parent.ID, "改名"); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, l).QuickAccess[1]; got.TargetID != parent.ID || got.ID != first.ID {
		t.Fatal("改名破坏固定项")
	}
	if err := l.DeleteFolder(t.Context(), parent.ID); err != nil {
		t.Fatal(err)
	}
	pins = snapshot(t, l).QuickAccess
	if len(pins) != 1 || pins[0].ID != third.ID {
		t.Fatal("级联删除未清理固定项", pins)
	}
	if err := l.DeleteTag(t.Context(), tag.ID); err != nil {
		t.Fatal(err)
	}
	if len(snapshot(t, l).QuickAccess) != 0 {
		t.Fatal("删标签残留固定项")
	}
}

func TestQuickAccessMoveFailureRollsBackWholeOrder(t *testing.T) {
	l := newTestLibrary(t)
	a, err := l.CreateFolder(t.Context(), "A", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := l.CreateFolder(t.Context(), "B", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := l.PinQuickAccess(t.Context(), "folder", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := l.PinQuickAccess(t.Context(), "folder", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, l).QuickAccess
	if _, err := l.DB.Exec("CREATE TRIGGER block_order BEFORE UPDATE ON quick_access WHEN OLD.id='" + first.ID + "' BEGIN SELECT RAISE(ABORT,'blocked'); END"); err != nil {
		t.Fatal(err)
	}
	if err := l.MoveQuickAccess(t.Context(), second.ID, 0); err == nil {
		t.Fatal("模拟排序失败未传出")
	}
	if got := snapshot(t, l).QuickAccess; !reflect.DeepEqual(got, before) {
		t.Fatal("排序失败仍修改部分行", got, before)
	}
}
