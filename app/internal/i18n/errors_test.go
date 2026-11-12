package i18n

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
)

func TestLocalizedErrorPreservesCauseAndLanguage(t *testing.T) {
	cause := &fs.PathError{Op: "open", Path: "photo.jpg", Err: fs.ErrPermission}
	err := WrapError(cause, "error_decode_image", "photo.jpg")
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatal("original error is no longer reachable")
	}
	var pathError *fs.PathError
	if !errors.As(err, &pathError) || pathError != cause {
		t.Fatal("original path diagnostics were lost")
	}
	var localized *LocalizedError
	if !errors.As(err, &localized) || localized.Key != "error_decode_image" {
		t.Fatal("localized error context is missing")
	}
	for _, tc := range []struct{ language, prefix string }{
		{"zh-CN", "解码图片 photo.jpg"},
		{"en", "Decode image photo.jpg"},
		{"ja", "画像photo.jpgをデコード"},
	} {
		text := (Catalog{Language: tc.language}).Error(err)
		if !strings.HasPrefix(text, tc.prefix) || !strings.Contains(text, cause.Error()) {
			t.Fatalf("%s diagnostics: %q", tc.language, text)
		}
	}
	if !strings.HasPrefix(err.Error(), "Decode image photo.jpg") {
		t.Fatalf("log diagnostics should be stable English: %q", err.Error())
	}
}

func TestLocalizedErrorPreservesNestedAndJoinedErrors(t *testing.T) {
	native := fmt.Errorf("database: %w", fs.ErrNotExist)
	err := WrapError(errors.Join(
		NewError("error_name_required"),
		WrapError(native, "error_database_invalid"),
	), "operation_failed")
	text := (Catalog{Language: "zh-CN"}).Error(err)
	for _, want := range []string{"操作失败", "名称不能为空", "图库数据库无效", native.Error()} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("joined native error chain was lost")
	}
}

func TestLocalizedErrorNativeAndNil(t *testing.T) {
	c := Catalog{Language: "zh-CN"}
	native := fmt.Errorf("open library: %w", fs.ErrNotExist)
	if got := c.Error(native); got != native.Error() {
		t.Fatalf("native diagnostics changed: %q", got)
	}
	if c.Error(nil) != "" || WrapError(nil, "operation_failed") != nil {
		t.Fatal("nil error should remain nil")
	}
}
