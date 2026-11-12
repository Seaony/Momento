package ui

import "github.com/egoist/mygo/ui"

var settingsIcons = map[string]*ui.SVG{
	"general": sidebarIcon("设置"),
	"browser": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i1" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i1" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i1-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill-rule="nonzero" fill="#000" /></mask></defs><clipPath id="gl-i1"><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" /></clipPath>
        <g filter="url(#sh-i1)"><g filter="url(#hi-i1)"><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill="url(#loS-i1)" /></g></g>
        <g clip-path="url(#gl-i1)" fill="none" stroke-width="1.6" stroke="#4A4B78">
          <path d="M3 12.5H28" /><path d="M10.9 12.5A4.6 11.5 0 1 0 20.1 12.5A4.6 11.5 0 1 0 10.9 12.5Z" />
        </g>
        <path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i1-0)" d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" /></svg>`)),
	"about": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i2" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i2" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i2-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i2)"><g filter="url(#hi-i2)"><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill="url(#loS-i2)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i2-0)" d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" /><path fill="#4A4B78" d="M14.2 7A1.3 1.3 0 1 0 16.8 7A1.3 1.3 0 1 0 14.2 7Z" /><path stroke="#4A4B78" d="M15.5 11V19" fill="none" stroke-width="2.2" stroke-linecap="round" /></svg>`)),
}

var settingsSymbols = map[string]*ui.SVG{
	"system":   ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="2.5" y="3.5" width="19" height="13" rx="1.5"/><path d="M12 16.5v4M8.5 20.5h7"/></svg>`)),
	"light":    ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 1v3M12 20v3M1 12h3M20 12h3M4.2 4.2l2.1 2.1M17.7 17.7l2.1 2.1M4.2 19.8l2.1-2.1M17.7 6.3l2.1-2.1"/></svg>`)),
	"dark":     ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M20.5 14.6A9 9 0 0 1 9.4 3.5a9 9 0 1 0 11.1 11.1Z" fill="#000"/></svg>`)),
	"chevrons": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 14" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M1 5 5 1 9 5M1 9 5 13 9 9"/></svg>`)),
}
