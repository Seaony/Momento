package i18n

import "strings"

type LocalizedError struct {
	Key   string
	Args  []any
	Cause error
}

func NewError(key string, args ...any) error {
	return &LocalizedError{Key: key, Args: args}
}

func WrapError(cause error, key string, args ...any) error {
	if cause == nil {
		return nil
	}
	return &LocalizedError{Key: key, Args: args, Cause: cause}
}

func (e *LocalizedError) Error() string {
	return (Catalog{Language: "en"}).Error(e)
}

func (e *LocalizedError) Unwrap() error { return e.Cause }

// 业务说明按界面语言显示，系统和数据库的原始诊断原样保留。
func (c Catalog) Error(err error) string {
	if err == nil {
		return ""
	}
	if localized, ok := err.(*LocalizedError); ok {
		text := c.T(localized.Key, localized.Args...)
		if localized.Cause != nil {
			text += ": " + c.Error(localized.Cause)
		}
		return text
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		parts := make([]string, 0, len(joined.Unwrap()))
		for _, cause := range joined.Unwrap() {
			if cause != nil {
				parts = append(parts, c.Error(cause))
			}
		}
		return strings.Join(parts, "\n")
	}
	return err.Error()
}
