package browser

import (
	"errors"
	"log/slog"
	"os"
)

const maxUploadBytesInFlight = 2 * MaxUploadSize

var (
	errUploadBudgetFull = errors.New("Upload storage is full. Try again after pending imports finish.")
	errUploadTooLarge   = errors.New("File exceeds 512 MB.")
)

type uploadFile struct {
	queue *jobQueue
	file  *os.File
	bytes int64
}

func (upload *uploadFile) Write(data []byte) (int, error) {
	size := int64(len(data))
	if size > MaxUploadSize-upload.bytes {
		return 0, errUploadTooLarge
	}
	queue := upload.queue
	queue.mu.Lock()
	if size > queue.uploadLimit-queue.uploadBytes {
		queue.mu.Unlock()
		return 0, errUploadBudgetFull
	}
	queue.uploadBytes += size
	upload.bytes += size
	queue.mu.Unlock()
	return upload.file.Write(data)
}

func (upload *uploadFile) release() {
	if upload.file == nil {
		return
	}
	if err := os.Remove(upload.file.Name()); err != nil && !os.IsNotExist(err) {
		slog.Warn("删除上传的临时文件失败", "error", err)
	}
	upload.queue.mu.Lock()
	upload.queue.uploadBytes -= upload.bytes
	upload.queue.mu.Unlock()
}
