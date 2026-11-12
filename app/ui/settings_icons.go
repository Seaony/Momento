package ui

import "github.com/egoist/mygo/ui"

var settingsIcons = map[string]*ui.SVG{
	"general": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" stroke="#000" stroke-width="2" stroke-linejoin="round"><path d="M12.026 4.34Q12.719 3.94 12.939 3.171L13.261 2.049Q13.481 1.28 14.281 1.28L16.719 1.28Q17.519 1.28 17.739 2.049L18.061 3.171Q18.281 3.94 18.974 4.34L20.829 5.412Q21.522 5.812 22.298 5.618L23.431 5.335Q24.207 5.142 24.607 5.835L25.826 7.946Q26.226 8.638 25.67 9.214L24.859 10.053Q24.303 10.629 24.303 11.429L24.303 13.571Q24.303 14.371 24.859 14.947L25.67 15.786Q26.226 16.362 25.826 17.054L24.607 19.165Q24.207 19.858 23.431 19.665L22.298 19.382Q21.522 19.188 20.829 19.588L18.974 20.66Q18.281 21.06 18.061 21.829L17.739 22.951Q17.519 23.72 16.719 23.72L14.281 23.72Q13.481 23.72 13.261 22.951L12.939 21.829Q12.719 21.06 12.026 20.66L10.171 19.588Q9.478 19.188 8.702 19.382L7.569 19.665Q6.793 19.858 6.393 19.165L5.174 17.054Q4.774 16.362 5.33 15.786L6.141 14.947Q6.697 14.371 6.697 13.571L6.697 11.429Q6.697 10.629 6.141 10.053L5.33 9.214Q4.774 8.638 5.174 7.946L6.393 5.835Q6.793 5.142 7.569 5.335L8.702 5.618Q9.478 5.812 10.171 5.412Z"/><circle cx="15.5" cy="12.5" r="4"/></svg>`)),
	"browser": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="1.6"><circle cx="12" cy="12" r="9"/><ellipse cx="12" cy="12" rx="4" ry="9"/><path d="M3 12h18"/></svg>`)),
	"about":   ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" fill="none" stroke="#000" stroke-width="1.6"/><circle cx="12" cy="7.5" r="1" fill="#000"/><path d="M12 11v6" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round"/></svg>`)),
	"appearance": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i0" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i0" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i0" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i0" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i0-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i0-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M6 12.5Q4 12.5 4.173 14.49A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 26.827 14.49Q27 12.5 25 12.5Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i0)"><g filter="url(#hi-i0)"><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill="url(#loS-i0)" /></g></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i0-0)" d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" />
      <g filter="url(#hi-i0)"><g transform="rotate(-90 15.5 12.5)"><path d="M6 12.5Q4 12.5 4.173 14.49A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 26.827 14.49Q27 12.5 25 12.5Z" fill="url(#upS-i0)" /></g></g>
      <g transform="rotate(-90 15.5 12.5)"><path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i0-1)" d="M6 12.5Q4 12.5 4.173 14.49A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 26.827 14.49Q27 12.5 25 12.5Z" /></g></svg>`)),
	"sidebar": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i1" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i1" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i1-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i1-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 7.9Q27 7.9 27 7.9L4 7.9Q4 7.9 4 7.9Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i1)"><g filter="url(#hi-i1)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#loS-i1)" /></g></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i1-0)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" />
      <path d="M6 11.1Q6 9.9 7.2 9.9L23.8 9.9Q25 9.9 25 11.1L25 18.5Q25 19.7 23.8 19.7L7.2 19.7Q6 19.7 6 18.5Z" fill="#fff" />
      <g opacity="0"><path d="M25 11.1Q25 9.9 25 9.9L25 9.9Q25 9.9 25 11.1L25 18.5Q25 19.7 25 19.7L25 19.7Q25 19.7 25 18.5Z" fill="#fff" /><path stroke-linejoin="round" stroke-linecap="round" stroke-width="1.6" fill="none" stroke="#2E3036" d="M26.8 12.5H26.8" /></g>
      <g filter="url(#hi-i1)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 7.9Q27 7.9 27 7.9L4 7.9Q4 7.9 4 7.9Z" fill="url(#upS-i1)" /></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i1-1)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 7.9Q27 7.9 27 7.9L4 7.9Q4 7.9 4 7.9Z" /></svg>`)),
	"controls": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i2" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i2" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i2-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i2)"><g filter="url(#hi-i2)"><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill="url(#loS-i2)" /></g></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i2-0)" d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" />
      <path d="M6.9 12.5A8.6 8.6 0 1 0 24.1 12.5A8.6 8.6 0 1 0 6.9 12.5Z" fill="#fff" />
      <g transform="rotate(40 15.5 12.5)">
        <path fill="#24202B" d="M13.1 12.5Q13.1 12.5 13.1 12.5L15.295 6.464Q15.5 5.9 15.705 6.464L17.9 12.5Q17.9 12.5 17.9 12.5Z" />
        <path fill="#707175" d="M17.9 12.5Q17.9 12.5 17.9 12.5L15.705 18.536Q15.5 19.1 15.295 18.536L13.1 12.5Q13.1 12.5 13.1 12.5Z" />
        <circle cx="15.5" cy="12.5" r="1.3" fill="#fff" />
      </g></svg>`)),
	"preview": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i3" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i3" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i3" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i3" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i3-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M-5.4 0A5.4 5.4 0 1 0 5.4 0A5.4 5.4 0 1 0 -5.4 0Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i3-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M3.6 12.5C7 6.6 11 4.2 15.5 4.2C20 4.2 24 6.6 27.4 12.5C24 18.4 20 20.8 15.5 20.8C11 20.8 7 18.4 3.6 12.5Z" fill-rule="nonzero" fill="#000" /></mask></defs><clippath id="eye-i3"><path d="M3.6 12.5C7 6.6 11 4.2 15.5 4.2C20 4.2 24 6.6 27.4 12.5C24 18.4 20 20.8 15.5 20.8C11 20.8 7 18.4 3.6 12.5Z" /></clippath>
        <g filter="url(#sh-i3)"><g filter="url(#hi-i3)"><path d="M3.6 12.5C7 6.6 11 4.2 15.5 4.2C20 4.2 24 6.6 27.4 12.5C24 18.4 20 20.8 15.5 20.8C11 20.8 7 18.4 3.6 12.5Z" fill="url(#loS-i3)" /></g></g>
        <g clip-path="url(#eye-i3)"><g transform="translate(15.5 12.5)"><g filter="url(#hi-i3)"><path d="M-5.4 0A5.4 5.4 0 1 0 5.4 0A5.4 5.4 0 1 0 -5.4 0Z" fill="url(#upS-i3)" /></g><path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i3-0)" d="M-5.4 0A5.4 5.4 0 1 0 5.4 0A5.4 5.4 0 1 0 -5.4 0Z" />
          <circle fill="#24202B" r="2.3" /></g></g>
        <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i3-1)" d="M3.6 12.5C7 6.6 11 4.2 15.5 4.2C20 4.2 24 6.6 27.4 12.5C24 18.4 20 20.8 15.5 20.8C11 20.8 7 18.4 3.6 12.5Z" /></svg>`)),
	"import": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i4" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i4" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i4" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i4" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i4-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i4-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M6.426 15.6Q4.426 15.6 5.129 17.47A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 25.871 17.47Q26.574 15.6 24.574 15.6Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i4)"><g filter="url(#hi-i4)"><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill="url(#loS-i4)" /></g></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i4-0)" d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" />
      <path d="M15.5 3V12.4M11.9 8.8L15.5 12.4L19.1 8.8" fill="none" stroke="#fff" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" />
      <g filter="url(#hi-i4)"><path d="M6.426 15.6Q4.426 15.6 5.129 17.47A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 25.871 17.47Q26.574 15.6 24.574 15.6Z" fill="url(#upS-i4)" /></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" d="M6.426 15.6Q4.426 15.6 5.129 17.47A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 25.871 17.47Q26.574 15.6 24.574 15.6Z" mask="url(#ol-i4-1)" /></svg>`)),
	"shortcuts": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i5" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i5" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i5" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i5" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i5-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M5.2 4.4Q5.2 2.2 7.4 2.2L23.6 2.2Q25.8 2.2 25.8 4.4L25.8 20.6Q25.8 22.8 23.6 22.8L7.4 22.8Q5.2 22.8 5.2 20.6Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i5)"><g filter="url(#hi-i5)"><path d="M5.2 4.4Q5.2 2.2 7.4 2.2L23.6 2.2Q25.8 2.2 25.8 4.4L25.8 20.6Q25.8 22.8 23.6 22.8L7.4 22.8Q5.2 22.8 5.2 20.6Z" fill="url(#loS-i5)" /></g></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i5-0)" d="M5.2 4.4Q5.2 2.2 7.4 2.2L23.6 2.2Q25.8 2.2 25.8 4.4L25.8 20.6Q25.8 22.8 23.6 22.8L7.4 22.8Q5.2 22.8 5.2 20.6Z" />
      <path d="M7.2 5.4Q7.2 4.2 8.4 4.2L13.3 4.2Q14.5 4.2 14.5 5.4L14.5 10.3Q14.5 11.5 13.3 11.5L8.4 11.5Q7.2 11.5 7.2 10.3Z" fill="#fff" /><path d="M7.2 14.7Q7.2 13.5 8.4 13.5L13.3 13.5Q14.5 13.5 14.5 14.7L14.5 19.6Q14.5 20.8 13.3 20.8L8.4 20.8Q7.2 20.8 7.2 19.6Z" fill="#fff" /><path d="M16.5 14.7Q16.5 13.5 17.7 13.5L22.6 13.5Q23.8 13.5 23.8 14.7L23.8 19.6Q23.8 20.8 22.6 20.8L17.7 20.8Q16.5 20.8 16.5 19.6Z" fill="#fff" />
      <g transform="translate(16.5 4.2) translate(3.65 3.65) scale(1) translate(-3.65 -3.65)"><g filter="url(#sh-i5)"><path d="M0 1.2Q0 0 1.2 0L6.1 0Q7.3 0 7.3 1.2L7.3 6.1Q7.3 7.3 6.1 7.3L1.2 7.3Q0 7.3 0 6.1Z" fill="#fff" /></g><circle fill="#2E3036" cx="3.65" cy="3.65" r="1.7" /></g></svg>`)),
	"notifications": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i7" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i7" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i7" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i7" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i7-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M7.8 18.4C9 17 9.3 15 9.3 12C9.3 8 12 5.2 15.5 5.2C19 5.2 21.7 8 21.7 12C21.7 15 22 17 23.2 18.4Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i7-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M5.6 19.3Q5.6 17.4 7.5 17.4L23.5 17.4Q25.4 17.4 25.4 19.3L25.4 19.3Q25.4 21.2 23.5 21.2L7.5 21.2Q5.6 21.2 5.6 19.3Z" fill-rule="nonzero" fill="#000" /></mask></defs><g transform="rotate(0 15.5 4.4)">
        <circle fill="#24202B" cx="15.5" cy="4.4" r="1.8" />
        <circle fill="#24202B" cx="15.5" cy="21.4" r="2.3" />
        <g filter="url(#sh-i7)"><g filter="url(#hi-i7)"><path d="M7.8 18.4C9 17 9.3 15 9.3 12C9.3 8 12 5.2 15.5 5.2C19 5.2 21.7 8 21.7 12C21.7 15 22 17 23.2 18.4Z" fill="url(#loS-i7)" /></g></g><path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i7-0)" d="M7.8 18.4C9 17 9.3 15 9.3 12C9.3 8 12 5.2 15.5 5.2C19 5.2 21.7 8 21.7 12C21.7 15 22 17 23.2 18.4Z" />
        <g filter="url(#hi-i7)"><path d="M5.6 19.3Q5.6 17.4 7.5 17.4L23.5 17.4Q25.4 17.4 25.4 19.3L25.4 19.3Q25.4 21.2 23.5 21.2L7.5 21.2Q5.6 21.2 5.6 19.3Z" fill="url(#upS-i7)" /></g><path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i7-1)" d="M5.6 19.3Q5.6 17.4 7.5 17.4L23.5 17.4Q25.4 17.4 25.4 19.3L25.4 19.3Q25.4 21.2 23.5 21.2L7.5 21.2Q5.6 21.2 5.6 19.3Z" />
      </g></svg>`)),
	"password": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none" aria-hidden="true"><defs><lineargradient id="loS-i6" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7C7D81" /><stop offset="1" stop-color="#636469" /></lineargradient><lineargradient id="upS-i6" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#C1C2C3" /><stop offset="1" stop-color="#9B9B9F" /></lineargradient>
  <filter id="sh-i6" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.443 0 0 0 0 0.451 0 0 0 0 0.463 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i6" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i6-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M5.6 13.2Q5.6 11 7.8 11L23.2 11Q25.4 11 25.4 13.2L25.4 21.4Q25.4 23.6 23.2 23.6L7.8 23.6Q5.6 23.6 5.6 21.4Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g transform="translate(0 0) rotate(0 20.5 12)" filter="url(#sh-i6)">
        <path stroke="#66686C" d="M10.5 12V7.6A5 5 0 0 1 20.5 7.6V12" fill="none" stroke-width="3.2" stroke-linecap="round" />
        <path d="M10.5 12V7.6A5 5 0 0 1 20.5 7.6V12" fill="none" stroke-width="1" stroke-linecap="round" stroke="#fff" stroke-opacity=".45" />
      </g>
      <g filter="url(#sh-i6)"><g filter="url(#hi-i6)"><path d="M5.6 13.2Q5.6 11 7.8 11L23.2 11Q25.4 11 25.4 13.2L25.4 21.4Q25.4 23.6 23.2 23.6L7.8 23.6Q5.6 23.6 5.6 21.4Z" fill="url(#upS-i6)" /></g></g>
      <path stroke-linejoin="round" stroke-linecap="round" stroke-width="2" fill="none" stroke="#2E3036" mask="url(#ol-i6-0)" d="M5.6 13.2Q5.6 11 7.8 11L23.2 11Q25.4 11 25.4 13.2L25.4 21.4Q25.4 23.6 23.2 23.6L7.8 23.6Q5.6 23.6 5.6 21.4Z" />
      <g fill="#24202B" transform="rotate(0 15.5 16.2)"><circle cx="15.5" cy="16.2" r="2.5" /><path d="M14.6 17.5Q14.6 16.6 15.5 16.6L15.5 16.6Q16.4 16.6 16.4 17.5L16.4 20.3Q16.4 21.2 15.5 21.2L15.5 21.2Q14.6 21.2 14.6 20.3Z" /></g></svg>`)),
}

var settingsSymbols = map[string]*ui.SVG{
	"chevrons": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 14" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M1 5 5 1 9 5M1 9 5 13 9 9"/></svg>`)),
}
