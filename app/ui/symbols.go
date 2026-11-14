package ui

import "github.com/egoist/mygo/ui"

var momentoSymbols = map[string]*ui.SVG{
	"rotate-left":      ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 10A8 8 0 1 1 6.3 18M4 4V10H10"/></svg>`)),
	"rotate-right":     ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><g transform="translate(24 0) scale(-1 1)"><path d="M4 10A8 8 0 1 1 6.3 18M4 4V10H10"/></g></svg>`)),
	"chevron-left":     ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18L9 12L15 6"/></svg>`)),
	"chevron-right":    ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18L15 12L9 6"/></svg>`)),
	"chevron-down":     ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9L12 15L18 9"/></svg>`)),
	"chevron-up":       ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 15L12 9L18 15"/></svg>`)),
	"chevrons-up-down": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M8 10L12 6L16 10M8 14L12 18L16 14"/></svg>`)),
	"plus":             ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5V19M5 12H19"/></svg>`)),
	"minus":            ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12H19"/></svg>`)),
	"close":            ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M6 6L18 18M18 6L6 18"/></svg>`)),
	"check":            ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12L10 17L19 7"/></svg>`)),
}

var appSymbolAliases = map[string]string{
	"back":     "chevron-left",
	"forward":  "chevron-right",
	"chevrons": "chevrons-up-down",
	"add":      "plus",
}

func appSymbol(c *ui.Context, name string) ui.Element {
	if canonical, ok := appSymbolAliases[name]; ok {
		name = canonical
	}
	return ui.Icon(c, momentoSymbols[name]).Size(14, 14)
}
