package main

import (
	"log/slog"
	"os"

	"github.com/Seaony/Momento/app/internal/logging"
	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
)

func main() {
	logs, err := logging.Open()
	if err != nil {
		slog.Error("无法创建日志", "error", err)
		os.Exit(1)
	}
	defer logs.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	slog.Info("启动", "version", "2.0.0", "logDirectory", logging.Directory())
	mygo.App.SetName("Momento")
	mygo.App.WhenReady(func() {
		app := appui.New()
		app.OpenWindow()
	})
	if err := mygo.App.Run(); err != nil {
		slog.Error("应用退出失败", "error", err)
		os.Exit(1)
	}
	slog.Info("退出")
}
