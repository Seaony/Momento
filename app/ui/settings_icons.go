package ui

import "github.com/egoist/mygo/ui"

var settingsIcons = map[string]*ui.SVG{
	"general": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" stroke="#000" stroke-width="2" stroke-linejoin="round"><path d="M12.026 4.34Q12.719 3.94 12.939 3.171L13.261 2.049Q13.481 1.28 14.281 1.28L16.719 1.28Q17.519 1.28 17.739 2.049L18.061 3.171Q18.281 3.94 18.974 4.34L20.829 5.412Q21.522 5.812 22.298 5.618L23.431 5.335Q24.207 5.142 24.607 5.835L25.826 7.946Q26.226 8.638 25.67 9.214L24.859 10.053Q24.303 10.629 24.303 11.429L24.303 13.571Q24.303 14.371 24.859 14.947L25.67 15.786Q26.226 16.362 25.826 17.054L24.607 19.165Q24.207 19.858 23.431 19.665L22.298 19.382Q21.522 19.188 20.829 19.588L18.974 20.66Q18.281 21.06 18.061 21.829L17.739 22.951Q17.519 23.72 16.719 23.72L14.281 23.72Q13.481 23.72 13.261 22.951L12.939 21.829Q12.719 21.06 12.026 20.66L10.171 19.588Q9.478 19.188 8.702 19.382L7.569 19.665Q6.793 19.858 6.393 19.165L5.174 17.054Q4.774 16.362 5.33 15.786L6.141 14.947Q6.697 14.371 6.697 13.571L6.697 11.429Q6.697 10.629 6.141 10.053L5.33 9.214Q4.774 8.638 5.174 7.946L6.393 5.835Q6.793 5.142 7.569 5.335L8.702 5.618Q9.478 5.812 10.171 5.412Z"/><circle cx="15.5" cy="12.5" r="4"/></svg>`)),
	"browser": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="1.6"><circle cx="12" cy="12" r="9"/><ellipse cx="12" cy="12" rx="4" ry="9"/><path d="M3 12h18"/></svg>`)),
	"about":   ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" fill="none" stroke="#000" stroke-width="1.6"/><circle cx="12" cy="7.5" r="1" fill="#000"/><path d="M12 11v6" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round"/></svg>`)),
}

var settingsSymbols = map[string]*ui.SVG{
	"system":   ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="2.5" y="3.5" width="19" height="13" rx="1.5"/><path d="M12 16.5v4M8.5 20.5h7"/></svg>`)),
	"light":    ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 1v3M12 20v3M1 12h3M20 12h3M4.2 4.2l2.1 2.1M17.7 17.7l2.1 2.1M4.2 19.8l2.1-2.1M17.7 6.3l2.1-2.1"/></svg>`)),
	"dark":     ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M20.5 14.6A9 9 0 0 1 9.4 3.5a9 9 0 1 0 11.1 11.1Z" fill="#000"/></svg>`)),
	"chevrons": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 14" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M1 5 5 1 9 5M1 9 5 13 9 9"/></svg>`)),
}
