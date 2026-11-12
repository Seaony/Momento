package ui

import (
	"github.com/Seaony/Momento/app/resources"
	"github.com/egoist/mygo/ui"
)

var appLogo = func() *ui.Bitmap {
	bitmap, err := ui.DecodeBitmap(resources.WelcomeIcon)
	if err != nil {
		panic(err)
	}
	return bitmap
}()

func (a *App) welcome(c *ui.Context) {
	ui.Column(c).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+16), 24, 24, 24).Center().Gap(16).DragWindow().Children(func() {
		ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
		ui.Text(c, a.T("welcome")).FontSize(24).Bold()
		ui.Text(c, a.T("welcome_hint")).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(12).Margin(12, 0).Children(func() {
			if ui.PrimaryButton(c, a.T("new_library")).Clicked() {
				a.ShowNewLibrary()
			}
			if ui.Button(c, a.T("open_library")).Clicked() {
				a.ChooseOpenLibrary()
			}
		})
		for _, recent := range a.Recent {
			if ui.Button(c.Key(recent.Path), recent.DisplayName).Width(320).Clicked() {
				a.OpenLibrary(recent.Path)
			}
		}
	})
}
