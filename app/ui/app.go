package ui

import (
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type App struct {
	Window *mygo.Window
	Texts  i18n.Catalog
}

func New() *App { return &App{Texts: i18n.Catalog{Language: "system"}} }

func (a *App) OpenWindow() {
	a.Window = mygo.NewWindow(mygo.WindowOptions{
		Title: "Momento", Width: 1100, Height: 800,
		MinWidth: 1100, MinHeight: 640, StateKey: "main",
		Content: ui.View(a.View),
	})
}

func (a *App) View(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(18).Children(func() {
		ui.Text(c, "Momento").FontSize(32).Bold()
		ui.Text(c, a.Texts.T("welcome")).FontSize(20).Bold()
		ui.Text(c, a.Texts.T("welcome_hint")).TextColor(c.Theme().TextMuted)
	})
}
