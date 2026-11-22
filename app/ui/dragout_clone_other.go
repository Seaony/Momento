//go:build !darwin

package ui

// 这些平台没有统一的写时复制接口，直接走普通复制。
func cloneDragFile(source, destination string) error { return errCloneUnsupported }
