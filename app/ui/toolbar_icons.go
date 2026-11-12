package ui

import "github.com/egoist/mygo/ui"

var toolbarIcons = map[string]*ui.SVG{
	"filter": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i7" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i7" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i7" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i7" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i7-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M6.156 3.883Q5.2 2.6 6.8 2.6L24.2 2.6Q25.8 2.6 24.844 3.883L18.678 12.158Q18.2 12.8 17.4 12.8L13.6 12.8Q12.8 12.8 12.322 12.158Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i7-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M12.8 12.4Q12.8 12 13.2 12L17.8 12Q18.2 12 18.2 12.4L18.2 20.4Q18.2 21.4 17.299 21.834L13.701 23.566Q12.8 24 12.8 23Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i7)"><g filter="url(#hi-i7)"><path d="M6.156 3.883Q5.2 2.6 6.8 2.6L24.2 2.6Q25.8 2.6 24.844 3.883L18.678 12.158Q18.2 12.8 17.4 12.8L13.6 12.8Q12.8 12.8 12.322 12.158Z" fill="url(#loS-i7)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i7-0)" d="M6.156 3.883Q5.2 2.6 6.8 2.6L24.2 2.6Q25.8 2.6 24.844 3.883L18.678 12.158Q18.2 12.8 17.4 12.8L13.6 12.8Q12.8 12.8 12.322 12.158Z" />
        <g filter="url(#sh-i7)"><g filter="url(#hi-i7)"><path d="M12.8 12.4Q12.8 12 13.2 12L17.8 12Q18.2 12 18.2 12.4L18.2 20.4Q18.2 21.4 17.299 21.834L13.701 23.566Q12.8 24 12.8 23Z" fill="url(#upS-i7)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i7-1)" d="M12.8 12.4Q12.8 12 13.2 12L17.8 12Q18.2 12 18.2 12.4L18.2 20.4Q18.2 21.4 17.299 21.834L13.701 23.566Q12.8 24 12.8 23Z" />
        <path stroke-linecap="round" stroke-width="1.6" fill="none" d="M9.3 5.8H21.7M12.2 8.8H18.8" stroke="#4A4B78" /></svg>`)),
	"sidebar": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i8" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i8" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i8" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i8" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i8-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i8-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 7.9Q27 7.9 27 7.9L4 7.9Q4 7.9 4 7.9Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i8)"><g filter="url(#hi-i8)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#loS-i8)" /></g></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i8-0)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" />
      <path d="M6 11.1Q6 9.9 7.2 9.9L23.8 9.9Q25 9.9 25 11.1L25 18.5Q25 19.7 23.8 19.7L7.2 19.7Q6 19.7 6 18.5Z" fill="#fff" />
      <g opacity="0"><path d="M25 11.1Q25 9.9 25 9.9L25 9.9Q25 9.9 25 11.1L25 18.5Q25 19.7 25 19.7L25 19.7Q25 19.7 25 18.5Z" fill="#fff" /><path stroke-linecap="round" stroke-width="1.6" fill="none" stroke="#5D7198" d="M26.8 12.5H26.8" /></g>
      <g filter="url(#hi-i8)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 7.9Q27 7.9 27 7.9L4 7.9Q4 7.9 4 7.9Z" fill="url(#upS-i8)" /></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i8-1)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 7.9Q27 7.9 27 7.9L4 7.9Q4 7.9 4 7.9Z" /></svg>`)),
	"inspector": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i9" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i9" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i9" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i9" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i9-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i9-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M0 1.6Q0 0 1.6 0L8 0Q9.6 0 9.6 1.6L9.6 5.4Q9.6 7 8 7L1.6 7Q0 7 0 5.4Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i9)"><g filter="url(#hi-i9)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#loS-i9)" /></g></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i9-0)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" />
      <path d="M6 6.5Q6 5.3 7.2 5.3L23.8 5.3Q25 5.3 25 6.5L25 18.5Q25 19.7 23.8 19.7L7.2 19.7Q6 19.7 6 18.5Z" fill="#fff" />
      <g transform="translate(14 10.6) scale(1)"><g filter="url(#sh-i9)"><g filter="url(#hi-i9)"><path d="M0 1.6Q0 0 1.6 0L8 0Q9.6 0 9.6 1.6L9.6 5.4Q9.6 7 8 7L1.6 7Q0 7 0 5.4Z" fill="url(#upS-i9)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i9-1)" d="M0 1.6Q0 0 1.6 0L8 0Q9.6 0 9.6 1.6L9.6 5.4Q9.6 7 8 7L1.6 7Q0 7 0 5.4Z" />
        <path fill="#4A4B78" d="M3.8 1.8L6.6 3.5L3.8 5.2Z" stroke-width="1" stroke-linejoin="round" /></g></svg>`)),
	"import": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i10" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i10" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i10" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i10" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i10-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i10-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M6.426 15.6Q4.426 15.6 5.129 17.47A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 25.871 17.47Q26.574 15.6 24.574 15.6Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i10)"><g filter="url(#hi-i10)"><path d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" fill="url(#loS-i10)" /></g></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i10-0)" d="M4 12.5A11.5 11.5 0 1 0 27 12.5A11.5 11.5 0 1 0 4 12.5Z" />
      <path d="M15.5 3V12.4M11.9 8.8L15.5 12.4L19.1 8.8" fill="none" stroke="#4A4B78" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" />
      <g filter="url(#hi-i10)"><path d="M6.426 15.6Q4.426 15.6 5.129 17.47A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 25.871 17.47Q26.574 15.6 24.574 15.6Z" fill="url(#upS-i10)" /></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" d="M6.426 15.6Q4.426 15.6 5.129 17.47A11.5 11.5 0 0 0 15.5 24L15.5 24A11.5 11.5 0 0 0 25.871 17.47Q26.574 15.6 24.574 15.6Z" mask="url(#ol-i10-1)" /></svg>`)),
	"sort": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i11" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i11" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i11" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i11" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i11-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i11)"><g filter="url(#hi-i11)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#loS-i11)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i11-0)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" /><path d="M7 7H16M7 12.5H13M7 18H10" stroke="#4A4B78" stroke-width="1.6" stroke-linecap="round" /><path d="M21 6.5V15" stroke="#4A4B78" stroke-width="2.2" stroke-linecap="round" /><path d="M18.039 14.821Q17.8 14.5 18.2 14.5L23.8 14.5Q24.2 14.5 23.961 14.821L21.239 18.479Q21 18.8 20.761 18.479Z" fill="#4A4B78" /></svg>`)),
	"masonry": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i12" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i12" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i12" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i12" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i12-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i12)"><g filter="url(#hi-i12)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#loS-i12)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i12-0)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" /><path d="M5.8 6.4Q5.8 5.2 7 5.2L12.7 5.2Q13.9 5.2 13.9 6.4L13.9 13.8Q13.9 15 12.7 15L7 15Q5.8 15 5.8 13.8Z" fill="#fff" /><path d="M16.9 6.4Q16.9 5.2 18.1 5.2L23.8 5.2Q25 5.2 25 6.4L25 9.4Q25 10.6 23.8 10.6L18.1 10.6Q16.9 10.6 16.9 9.4Z" fill="#fff" /><path d="M5.8 19Q5.8 17.8 7 17.8L12.7 17.8Q13.9 17.8 13.9 19L13.9 20.4Q13.9 21.6 12.7 21.6L7 21.6Q5.8 21.6 5.8 20.4Z" fill="#fff" /><path d="M16.9 14.6Q16.9 13.4 18.1 13.4L23.8 13.4Q25 13.4 25 14.6L25 20.4Q25 21.6 23.8 21.6L18.1 21.6Q16.9 21.6 16.9 20.4Z" fill="#fff" /></svg>`)),
	"grid": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i13" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i13" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i13" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i13" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i13-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i13)"><g filter="url(#hi-i13)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#loS-i13)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i13-0)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" /><path d="M5.8 6.4Q5.8 5.2 7 5.2L12.7 5.2Q13.9 5.2 13.9 6.4L13.9 10.6Q13.9 11.8 12.7 11.8L7 11.8Q5.8 11.8 5.8 10.6Z" fill="#fff" /><path d="M16.9 6.4Q16.9 5.2 18.1 5.2L23.8 5.2Q25 5.2 25 6.4L25 10.6Q25 11.8 23.8 11.8L18.1 11.8Q16.9 11.8 16.9 10.6Z" fill="#fff" /><path d="M5.8 16Q5.8 14.8 7 14.8L12.7 14.8Q13.9 14.8 13.9 16L13.9 20.2Q13.9 21.4 12.7 21.4L7 21.4Q5.8 21.4 5.8 20.2Z" fill="#fff" /><path d="M16.9 16Q16.9 14.8 18.1 14.8L23.8 14.8Q25 14.8 25 16L25 20.2Q25 21.4 23.8 21.4L18.1 21.4Q16.9 21.4 16.9 20.2Z" fill="#fff" /></svg>`)),
	"list": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i14" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i14" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i14" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i14" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i14-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i14)"><g filter="url(#hi-i14)"><path d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#loS-i14)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i14-0)" d="M4 5.5Q4 3.3 6.2 3.3L24.8 3.3Q27 3.3 27 5.5L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" /><path d="M5.8 6.4Q5.8 5.2 7 5.2L23.8 5.2Q25 5.2 25 6.4L25 7.2Q25 8.4 23.8 8.4L7 8.4Q5.8 8.4 5.8 7.2Z" fill="#fff" /><path d="M5.8 13Q5.8 11.8 7 11.8L23.8 11.8Q25 11.8 25 13L25 13.8Q25 15 23.8 15L7 15Q5.8 15 5.8 13.8Z" fill="#fff" /><path d="M5.8 19.6Q5.8 18.4 7 18.4L23.8 18.4Q25 18.4 25 19.6L25 20.4Q25 21.6 23.8 21.6L7 21.6Q5.8 21.6 5.8 20.4Z" fill="#fff" /></svg>`)),
}

func appIcon(c *ui.Context, name string) ui.Element {
	var icon *ui.SVG
	switch name {
	case "heart", "favorite":
		icon = sidebarIcon("喜欢")
	case "folder":
		icon = sidebarIcon("文件夹")
	case "tags":
		icon = sidebarIcon("标签组")
	case "tag":
		icon = sidebarIcon("标签")
	case "trash":
		icon = sidebarIcon("回收站")
	case "settings":
		icon = sidebarIcon("设置")
	case "photos", "preview":
		icon = sidebarIcon("照片")
	case "menu":
		icon = toolbarIcons["list"]
	default:
		icon = toolbarIcons[name]
	}
	return ui.Image(c, icon).Size(18, 18)
}
