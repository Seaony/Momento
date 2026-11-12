package ui

import "github.com/egoist/mygo/ui"

var sidebarIcons = map[string]*ui.SVG{
	"照片": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i0" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i0" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient><linearGradient id="lo-i0-920" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="0.92" stop-color="#A0B0CE" /><stop offset="0.92" stop-color="#54678A" /><stop offset="1" stop-color="#B0BED6" /></linearGradient><linearGradient id="pure-i0-5393FF" x1="0" y1="0" x2="0" y2="1"><stop stop-color="#D2D7E2" offset="0" /><stop stop-color="#5D7198" offset="1" /></linearGradient>
  <filter id="sh-i0" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i0" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i0-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M6 5.5Q6 3.3 8.2 3.3L22.8 3.3Q25 3.3 25 5.5L25 16.1Q25 18.3 22.8 18.3L8.2 18.3Q6 18.3 6 16.1Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i0-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 7.8Q4 5.6 6.2 5.6L24.8 5.6Q27 5.6 27 7.8L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g transform="translate(0 0) rotate(0 15.5 21.7)"><g filter="url(#sh-i0)"><g filter="url(#hi-i0)"><path d="M6 5.5Q6 3.3 8.2 3.3L22.8 3.3Q25 3.3 25 5.5L25 16.1Q25 18.3 22.8 18.3L8.2 18.3Q6 18.3 6 16.1Z" fill="url(#lo-i0-920)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i0-0)" d="M6 5.5Q6 3.3 8.2 3.3L22.8 3.3Q25 3.3 25 5.5L25 16.1Q25 18.3 22.8 18.3L8.2 18.3Q6 18.3 6 16.1Z" /></g>
      <g transform="translate(0 0) rotate(0 15.5 21.7)">
        <clipPath id="sky-i0"><path d="M4 7.8Q4 5.6 6.2 5.6L24.8 5.6Q27 5.6 27 7.8L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" /></clipPath>
        <g filter="url(#sh-i0)"><g filter="url(#hi-i0)"><path d="M4 7.8Q4 5.6 6.2 5.6L24.8 5.6Q27 5.6 27 7.8L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" fill="url(#upS-i0)" /></g></g>
        <g clip-path="url(#sky-i0)">
          <circle cx="20.4" cy="9.8" r="2.5" fill="#fff" />
          <path d="M-1 23Q-1 23 -1 23L-1 20.2Q-1 20.2 -1 20.2L9.202 11.933Q10.6 10.8 12.001 11.93L23 20.8Q23 20.8 23 20.8L23 23Q23 23 23 23Z" fill="url(#loS-i0)" />
          <path d="M8 23Q8 23 8 23L8 21.4Q8 21.4 8 21.4L18.322 14.227Q19.8 13.2 21.328 14.152L32 20.8Q32 20.8 32 20.8L32 23Q32 23 32 23Z" fill="url(#pure-i0-5393FF)" />
        </g>
        <path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i0-1)" d="M4 7.8Q4 5.6 6.2 5.6L24.8 5.6Q27 5.6 27 7.8L27 19.5Q27 21.7 24.8 21.7L6.2 21.7Q4 21.7 4 19.5Z" />
      </g></svg>`)),
	"喜欢": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
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
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i1-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M0 9.6C-5.5 5.6 -10.9 1.4 -10.9 -3.6C-10.9 -7 -8.3 -9.6 -5.1 -9.6C-2.9 -9.6 -1.1 -8.4 0 -6.6C1.1 -8.4 2.9 -9.6 5.1 -9.6C8.3 -9.6 10.9 -7 10.9 -3.6C10.9 1.4 5.5 5.6 0 9.6Z" fill-rule="nonzero" fill="#000" /></mask></defs><g transform="translate(15.5 13.2) scale(1)">
        <g filter="url(#sh-i1)"><g filter="url(#hi-i1)"><path d="M0 9.6C-5.5 5.6 -10.9 1.4 -10.9 -3.6C-10.9 -7 -8.3 -9.6 -5.1 -9.6C-2.9 -9.6 -1.1 -8.4 0 -6.6C1.1 -8.4 2.9 -9.6 5.1 -9.6C8.3 -9.6 10.9 -7 10.9 -3.6C10.9 1.4 5.5 5.6 0 9.6Z" fill="url(#loS-i1)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i1-0)" d="M0 9.6C-5.5 5.6 -10.9 1.4 -10.9 -3.6C-10.9 -7 -8.3 -9.6 -5.1 -9.6C-2.9 -9.6 -1.1 -8.4 0 -6.6C1.1 -8.4 2.9 -9.6 5.1 -9.6C8.3 -9.6 10.9 -7 10.9 -3.6C10.9 1.4 5.5 5.6 0 9.6Z" />
        <g transform="translate(0 -0.6) scale(0.5)"><g filter="url(#hi-i1)"><path d="M0 9.6C-5.5 5.6 -10.9 1.4 -10.9 -3.6C-10.9 -7 -8.3 -9.6 -5.1 -9.6C-2.9 -9.6 -1.1 -8.4 0 -6.6C1.1 -8.4 2.9 -9.6 5.1 -9.6C8.3 -9.6 10.9 -7 10.9 -3.6C10.9 1.4 5.5 5.6 0 9.6Z" fill="url(#upS-i1)" /></g></g>
      </g></svg>`)),
	"文件夹": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient><linearGradient id="lo-i2-943" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="0.943" stop-color="#A0B0CE" /><stop offset="0.943" stop-color="#54678A" /><stop offset="1" stop-color="#B0BED6" /></linearGradient><linearGradient id="up-i2-933" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="0.933" stop-color="#C5CEE1" /><stop offset="0.933" stop-color="#A6B5D1" /><stop offset="1" stop-color="#B0BED6" /></linearGradient>
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
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i2-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 4.2Q4 2 6.2 2L9.8 2Q11.2 2 12.19 2.99L13.21 4.01Q14.2 5 15.6 5L24.8 5Q27 5 27 7.2L27 20.8Q27 23 24.8 23L6.2 23Q4 23 4 20.8Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i2-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 7.2Q4 5 6.2 5L24.8 5Q27 5 27 7.2L27 20.8Q27 23 24.8 23L6.2 23Q4 23 4 20.8Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i2)"><path d="M4 4.2Q4 2 6.2 2L9.8 2Q11.2 2 12.19 2.99L13.21 4.01Q14.2 5 15.6 5L24.8 5Q27 5 27 7.2L27 20.8Q27 23 24.8 23L6.2 23Q4 23 4 20.8Z" fill="url(#lo-i2-943)" /></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" d="M4 4.2Q4 2 6.2 2L9.8 2Q11.2 2 12.19 2.99L13.21 4.01Q14.2 5 15.6 5L24.8 5Q27 5 27 7.2L27 20.8Q27 23 24.8 23L6.2 23Q4 23 4 20.8Z" mask="url(#ol-i2-0)" />
      <g transform="translate(7 9)"><path d="M0 1Q0 0 1 0L16 0Q17 0 17 1L17 8Q17 9 16 9L1 9Q0 9 0 8Z" fill="#fff" /><path stroke-linecap="round" fill="none" stroke="#DEE2EE" d="M5.5 2.3H11.5" stroke-width="1.6" /></g>
      <g filter="url(#hi-i2)"><path d="M4 7.2Q4 5 6.2 5L24.8 5Q27 5 27 7.2L27 20.8Q27 23 24.8 23L6.2 23Q4 23 4 20.8Z" fill="url(#up-i2-933)" /></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" d="M4 7.2Q4 5 6.2 5L24.8 5Q27 5 27 7.2L27 20.8Q27 23 24.8 23L6.2 23Q4 23 4 20.8Z" mask="url(#ol-i2-1)" /></svg>`)),
	"标签组": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i3" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i3" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient><linearGradient id="lo-i3-912" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="0.912" stop-color="#A0B0CE" /><stop offset="0.912" stop-color="#54678A" /><stop offset="1" stop-color="#B0BED6" /></linearGradient><linearGradient id="up-i3-912" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="0.912" stop-color="#C5CEE1" /><stop offset="0.912" stop-color="#A6B5D1" /><stop offset="1" stop-color="#B0BED6" /></linearGradient>
  <filter id="sh-i3" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i3" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i3-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M0 2.2Q0 0 2.2 0L15.8 0Q18 0 18 2.2L18 11.4Q18 13.6 15.8 13.6L2.2 13.6Q0 13.6 0 11.4Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i3-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M0 2.2Q0 0 2.2 0L15.8 0Q18 0 18 2.2L18 11.4Q18 13.6 15.8 13.6L2.2 13.6Q0 13.6 0 11.4Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g transform="translate(8.4 3.3) rotate(0 9 6.8)"><g filter="url(#sh-i3)"><g filter="url(#hi-i3)"><path d="M0 2.2Q0 0 2.2 0L15.8 0Q18 0 18 2.2L18 11.4Q18 13.6 15.8 13.6L2.2 13.6Q0 13.6 0 11.4Z" fill="url(#lo-i3-912)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i3-0)" d="M0 2.2Q0 0 2.2 0L15.8 0Q18 0 18 2.2L18 11.4Q18 13.6 15.8 13.6L2.2 13.6Q0 13.6 0 11.4Z" /></g>
      <g transform="translate(4.6 8.1) rotate(0 9 6.8)"><g filter="url(#sh-i3)"><g filter="url(#hi-i3)"><path d="M0 2.2Q0 0 2.2 0L15.8 0Q18 0 18 2.2L18 11.4Q18 13.6 15.8 13.6L2.2 13.6Q0 13.6 0 11.4Z" fill="url(#up-i3-912)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i3-1)" d="M0 2.2Q0 0 2.2 0L15.8 0Q18 0 18 2.2L18 11.4Q18 13.6 15.8 13.6L2.2 13.6Q0 13.6 0 11.4Z" />
        <path fill="#4A4B78" d="M2.4 3.8Q2.4 2.6 3.6 2.6L7.6 2.6Q8.8 2.6 8.8 3.8L8.8 3.8Q8.8 5 7.6 5L3.6 5Q2.4 5 2.4 3.8Z" />
        <path stroke-linecap="round" stroke-width="1.6" fill="none" stroke="#4A4B78" d="M2.4 8.4H14" stroke-opacity=".45" /></g></svg>`)),
	"标签": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i4" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i4" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i4" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i4" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i4-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M5.473 13.687Q4.4 12.5 5.473 11.313L9.927 6.387Q11 5.2 12.6 5.2L23.8 5.2Q26 5.2 26 7.4L26 17.6Q26 19.8 23.8 19.8L12.6 19.8Q11 19.8 9.927 18.613Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i4)"><g filter="url(#hi-i4)"><g transform="rotate(-34 15.5 12.5)"><path d="M5.473 13.687Q4.4 12.5 5.473 11.313L9.927 6.387Q11 5.2 12.6 5.2L23.8 5.2Q26 5.2 26 7.4L26 17.6Q26 19.8 23.8 19.8L12.6 19.8Q11 19.8 9.927 18.613Z" fill="url(#loS-i4)" /><circle cx="11.6" cy="12.5" r="2.2" fill="#fff" /></g></g></g>
        <g transform="rotate(-34 15.5 12.5)"><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i4-0)" d="M5.473 13.687Q4.4 12.5 5.473 11.313L9.927 6.387Q11 5.2 12.6 5.2L23.8 5.2Q26 5.2 26 7.4L26 17.6Q26 19.8 23.8 19.8L12.6 19.8Q11 19.8 9.927 18.613Z" /></g></svg>`)),
	"回收站": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i5" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i5" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient><linearGradient id="lo-i5-935" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="0.935" stop-color="#A0B0CE" /><stop offset="0.935" stop-color="#54678A" /><stop offset="1" stop-color="#B0BED6" /></linearGradient><linearGradient id="up-i5-931" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="0.931" stop-color="#C5CEE1" /><stop offset="0.931" stop-color="#A6B5D1" /><stop offset="1" stop-color="#B0BED6" /></linearGradient>
  <filter id="sh-i5" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i5" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i5-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M6.91 7.195Q6.8 6 8 6L23 6Q24.2 6 24.09 7.195L22.856 20.612Q22.6 23.4 19.8 23.4L11.2 23.4Q8.4 23.4 8.144 20.612Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i5-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M6.4 6.6Q5.4 6.6 5.4 5.6L5.4 5Q5.4 4 6.4 4L11.9 4Q12.4 4 12.4 3.5L12.4 2.4Q12.4 1.6 13.2 1.6L17.8 1.6Q18.6 1.6 18.6 2.4L18.6 3.5Q18.6 4 19.1 4L24.6 4Q25.6 4 25.6 5L25.6 5.6Q25.6 6.6 24.6 6.6Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i5)"><path d="M7.439 5.399Q7.4 4.8 8 4.8L23 4.8Q23.6 4.8 23.561 5.399L22.58 20.606Q22.4 23.4 19.6 23.4L11.4 23.4Q8.6 23.4 8.42 20.606Z" fill="url(#lo-i5-935)" /></g>
      <g transform="translate(10.3 12) rotate(0 5.2 3.8)" filter="url(#sh-i5)"><path d="M0 1.2Q0 0 1.2 0L9.2 0Q10.4 0 10.4 1.2L10.4 6.4Q10.4 7.6 9.2 7.6L1.2 7.6Q0 7.6 0 6.4Z" fill="url(#upS-i5)" /><circle cx="3" cy="3" r="1.4" fill="#fff" /></g>
      <g filter="url(#hi-i5)"><path d="M6.91 7.195Q6.8 6 8 6L23 6Q24.2 6 24.09 7.195L22.856 20.612Q22.6 23.4 19.8 23.4L11.2 23.4Q8.4 23.4 8.144 20.612Z" fill="url(#up-i5-931)" /></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" d="M6.91 7.195Q6.8 6 8 6L23 6Q24.2 6 24.09 7.195L22.856 20.612Q22.6 23.4 19.8 23.4L11.2 23.4Q8.4 23.4 8.144 20.612Z" mask="url(#ol-i5-0)" />
      <path stroke="#5D7198" d="M13.064 9.4L13.512 20.2M17.936 9.4L17.488 20.2" fill="none" stroke-width="1.6" stroke-linecap="round" stroke-opacity=".6" />
      <g transform="translate(0 0) rotate(0 15.5 5.3)"><g filter="url(#hi-i5)"><path d="M6.4 6.6Q5.4 6.6 5.4 5.6L5.4 5Q5.4 4 6.4 4L11.9 4Q12.4 4 12.4 3.5L12.4 2.4Q12.4 1.6 13.2 1.6L17.8 1.6Q18.6 1.6 18.6 2.4L18.6 3.5Q18.6 4 19.1 4L24.6 4Q25.6 4 25.6 5L25.6 5.6Q25.6 6.6 24.6 6.6Z" fill="url(#upS-i5)" /></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i5-1)" d="M6.4 6.6Q5.4 6.6 5.4 5.6L5.4 5Q5.4 4 6.4 4L11.9 4Q12.4 4 12.4 3.5L12.4 2.4Q12.4 1.6 13.2 1.6L17.8 1.6Q18.6 1.6 18.6 2.4L18.6 3.5Q18.6 4 19.1 4L24.6 4Q25.6 4 25.6 5L25.6 5.6Q25.6 6.6 24.6 6.6Z" /></g></svg>`)),
	"设置": ui.MustParseSVG([]byte(`<svg height="32" width="32" xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i6" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i6" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i6" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <fegaussianblur in="SourceAlpha" stdDeviation="2" /><feoffset dy="2" result="o" />
    <fecolormatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <femerge><femergenode /><femergenode in="SourceGraphic" /></femerge>
  </filter>
  <filter id="hi-i6" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <fecolormatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feoffset in="a" dy="0.7" /><fegaussianblur stdDeviation="0.35" />
    <fecomposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <fecolormatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feblend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i6-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M12.026 4.34Q12.719 3.94 12.939 3.171L13.261 2.049Q13.481 1.28 14.281 1.28L16.719 1.28Q17.519 1.28 17.739 2.049L18.061 3.171Q18.281 3.94 18.974 4.34L20.829 5.412Q21.522 5.812 22.298 5.618L23.431 5.335Q24.207 5.142 24.607 5.835L25.826 7.946Q26.226 8.638 25.67 9.214L24.859 10.053Q24.303 10.629 24.303 11.429L24.303 13.571Q24.303 14.371 24.859 14.947L25.67 15.786Q26.226 16.362 25.826 17.054L24.607 19.165Q24.207 19.858 23.431 19.665L22.298 19.382Q21.522 19.188 20.829 19.588L18.974 20.66Q18.281 21.06 18.061 21.829L17.739 22.951Q17.519 23.72 16.719 23.72L14.281 23.72Q13.481 23.72 13.261 22.951L12.939 21.829Q12.719 21.06 12.026 20.66L10.171 19.588Q9.478 19.188 8.702 19.382L7.569 19.665Q6.793 19.858 6.393 19.165L5.174 17.054Q4.774 16.362 5.33 15.786L6.141 14.947Q6.697 14.371 6.697 13.571L6.697 11.429Q6.697 10.629 6.141 10.053L5.33 9.214Q4.774 8.638 5.174 7.946L6.393 5.835Q6.793 5.142 7.569 5.335L8.702 5.618Q9.478 5.812 10.171 5.412Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i6-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M12 12.5A3.5 3.5 0 1 0 19 12.5A3.5 3.5 0 1 0 12 12.5Z" fill-rule="nonzero" fill="#000" /></mask></defs>
      <g filter="url(#sh-i6)"><g filter="url(#hi-i6)"><g transform="rotate(0 15.5 12.5)"><path d="M12.026 4.34Q12.719 3.94 12.939 3.171L13.261 2.049Q13.481 1.28 14.281 1.28L16.719 1.28Q17.519 1.28 17.739 2.049L18.061 3.171Q18.281 3.94 18.974 4.34L20.829 5.412Q21.522 5.812 22.298 5.618L23.431 5.335Q24.207 5.142 24.607 5.835L25.826 7.946Q26.226 8.638 25.67 9.214L24.859 10.053Q24.303 10.629 24.303 11.429L24.303 13.571Q24.303 14.371 24.859 14.947L25.67 15.786Q26.226 16.362 25.826 17.054L24.607 19.165Q24.207 19.858 23.431 19.665L22.298 19.382Q21.522 19.188 20.829 19.588L18.974 20.66Q18.281 21.06 18.061 21.829L17.739 22.951Q17.519 23.72 16.719 23.72L14.281 23.72Q13.481 23.72 13.261 22.951L12.939 21.829Q12.719 21.06 12.026 20.66L10.171 19.588Q9.478 19.188 8.702 19.382L7.569 19.665Q6.793 19.858 6.393 19.165L5.174 17.054Q4.774 16.362 5.33 15.786L6.141 14.947Q6.697 14.371 6.697 13.571L6.697 11.429Q6.697 10.629 6.141 10.053L5.33 9.214Q4.774 8.638 5.174 7.946L6.393 5.835Q6.793 5.142 7.569 5.335L8.702 5.618Q9.478 5.812 10.171 5.412Z" fill="url(#loS-i6)" /></g></g></g><g transform="rotate(0 15.5 12.5)"><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i6-0)" d="M12.026 4.34Q12.719 3.94 12.939 3.171L13.261 2.049Q13.481 1.28 14.281 1.28L16.719 1.28Q17.519 1.28 17.739 2.049L18.061 3.171Q18.281 3.94 18.974 4.34L20.829 5.412Q21.522 5.812 22.298 5.618L23.431 5.335Q24.207 5.142 24.607 5.835L25.826 7.946Q26.226 8.638 25.67 9.214L24.859 10.053Q24.303 10.629 24.303 11.429L24.303 13.571Q24.303 14.371 24.859 14.947L25.67 15.786Q26.226 16.362 25.826 17.054L24.607 19.165Q24.207 19.858 23.431 19.665L22.298 19.382Q21.522 19.188 20.829 19.588L18.974 20.66Q18.281 21.06 18.061 21.829L17.739 22.951Q17.519 23.72 16.719 23.72L14.281 23.72Q13.481 23.72 13.261 22.951L12.939 21.829Q12.719 21.06 12.026 20.66L10.171 19.588Q9.478 19.188 8.702 19.382L7.569 19.665Q6.793 19.858 6.393 19.165L5.174 17.054Q4.774 16.362 5.33 15.786L6.141 14.947Q6.697 14.371 6.697 13.571L6.697 11.429Q6.697 10.629 6.141 10.053L5.33 9.214Q4.774 8.638 5.174 7.946L6.393 5.835Q6.793 5.142 7.569 5.335L8.702 5.618Q9.478 5.812 10.171 5.412Z" /></g>
      <path d="M9.7 12.5A5.8 5.8 0 1 0 21.3 12.5A5.8 5.8 0 1 0 9.7 12.5Z" fill="#fff" />
      <g filter="url(#hi-i6)"><path d="M12 12.5A3.5 3.5 0 1 0 19 12.5A3.5 3.5 0 1 0 12 12.5Z" fill="url(#upS-i6)" /></g>
      <path stroke-width="2" fill="none" stroke="#5D7198" d="M12 12.5A3.5 3.5 0 1 0 19 12.5A3.5 3.5 0 1 0 12 12.5Z" mask="url(#ol-i6-1)" /></svg>`)),
}

func sidebarIcon(name string) *ui.SVG {
	if name == "随机浏览" {
		return browseRandomIcon
	}
	return sidebarIcons[name]
}

var browseRandomIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i0" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i0" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i0" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <feGaussianBlur in="SourceAlpha" stdDeviation="2" /><feOffset dy="2" result="o" />
    <feColorMatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <feMerge><feMergeNode /><feMergeNode in="SourceGraphic" /></feMerge>
  </filter>
  <filter id="hi-i0" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feOffset in="a" dy="0.7" /><feGaussianBlur stdDeviation="0.35" />
    <feComposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <feColorMatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feBlend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i0-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M4 4.4Q4 2.2 6.2 2.2L20.2 2.2Q22.4 2.2 22.4 4.4L22.4 18.4Q22.4 20.6 20.2 20.6L6.2 20.6Q4 20.6 4 18.4Z" fill-rule="nonzero" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i0-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M9.8 9.2Q9.8 7 12 7L24.8 7Q27 7 27 9.2L27 22Q27 24.2 24.8 24.2L12 24.2Q9.8 24.2 9.8 22Z" fill-rule="nonzero" fill="#000" /></mask></defs><g filter="url(#sh-i0)"><g filter="url(#hi-i0)"><path d="M4 4.4Q4 2.2 6.2 2.2L20.2 2.2Q22.4 2.2 22.4 4.4L22.4 18.4Q22.4 20.6 20.2 20.6L6.2 20.6Q4 20.6 4 18.4Z" fill="url(#loS-i0)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i0-0)" d="M4 4.4Q4 2.2 6.2 2.2L20.2 2.2Q22.4 2.2 22.4 4.4L22.4 18.4Q22.4 20.6 20.2 20.6L6.2 20.6Q4 20.6 4 18.4Z" />
      <circle cx="9.8" cy="7.6" r="1.4" fill="#fff" />
      <g filter="url(#sh-i0)"><g filter="url(#hi-i0)"><path d="M9.8 9.2Q9.8 7 12 7L24.8 7Q27 7 27 9.2L27 22Q27 24.2 24.8 24.2L12 24.2Q9.8 24.2 9.8 22Z" fill="url(#upS-i0)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i0-1)" d="M9.8 9.2Q9.8 7 12 7L24.8 7Q27 7 27 9.2L27 22Q27 24.2 24.8 24.2L12 24.2Q9.8 24.2 9.8 22Z" />
      <circle fill="#4A4B78" cx="14.2" cy="11.4" r="1.4" /><circle fill="#4A4B78" cx="22.6" cy="11.4" r="1.4" /><circle fill="#4A4B78" cx="18.4" cy="15.6" r="1.4" /><circle fill="#4A4B78" cx="14.2" cy="19.8" r="1.4" /><circle fill="#4A4B78" cx="22.6" cy="19.8" r="1.4" /></svg>`))
var browseLinkIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -3 31 31" fill="none"><defs><linearGradient id="loS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ADBBD5" /><stop offset="1" stop-color="#9EAECD" /></linearGradient><linearGradient id="upS-i1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#D8DFEB" /><stop offset="1" stop-color="#C1CBDF" /></linearGradient>
  <filter id="sh-i1" x="-40%" y="-40%" width="180%" height="190%" color-interpolation-filters="sRGB">
    <feGaussianBlur in="SourceAlpha" stdDeviation="2" /><feOffset dy="2" result="o" />
    <feColorMatrix in="o" type="matrix" values="0 0 0 0 0.655 0 0 0 0 0.714 0 0 0 0 0.82 0 0 0 0.21 0" />
    <feMerge><feMergeNode /><feMergeNode in="SourceGraphic" /></feMerge>
  </filter>
  <filter id="hi-i1" x="-10%" y="-10%" width="120%" height="130%" color-interpolation-filters="sRGB">
    <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="a" />
    <feOffset in="a" dy="0.7" /><feGaussianBlur stdDeviation="0.35" />
    <feComposite in2="a" operator="arithmetic" k2="-1" k3="1" />
    <feColorMatrix type="matrix" values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.5 0" />
    <feBlend in2="SourceGraphic" mode="overlay" />
  </filter><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i1-0"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M-7.8 0Q-7.8 -4.8 -3 -4.8L3 -4.8Q7.8 -4.8 7.8 0L7.8 0Q7.8 4.8 3 4.8L-3 4.8Q-7.8 4.8 -7.8 0ZM-5 0Q-5 -2 -3 -2L3 -2Q5 -2 5 0L5 0Q5 2 3 2L-3 2Q-5 2 -5 0Z" fill-rule="evenodd" fill="#000" /></mask><mask height="109" width="111" y="-40" x="-40" maskUnits="userSpaceOnUse" id="ol-i1-1"><rect x="-40" y="-40" width="111" height="109" fill="#fff"></rect><path d="M-7.8 0Q-7.8 -4.8 -3 -4.8L3 -4.8Q7.8 -4.8 7.8 0L7.8 0Q7.8 4.8 3 4.8L-3 4.8Q-7.8 4.8 -7.8 0ZM-5 0Q-5 -2 -3 -2L3 -2Q5 -2 5 0L5 0Q5 2 3 2L-3 2Q-5 2 -5 0Z" fill-rule="evenodd" fill="#000" /></mask></defs>
      <g transform="rotate(-45 15.5 12.5)">
        <g transform="translate(11.3 12.5)"><g filter="url(#sh-i1)"><g filter="url(#hi-i1)"><path d="M-7.8 0Q-7.8 -4.8 -3 -4.8L3 -4.8Q7.8 -4.8 7.8 0L7.8 0Q7.8 4.8 3 4.8L-3 4.8Q-7.8 4.8 -7.8 0ZM-5 0Q-5 -2 -3 -2L3 -2Q5 -2 5 0L5 0Q5 2 3 2L-3 2Q-5 2 -5 0Z" fill-rule="evenodd" fill="url(#loS-i1)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i1-0)" d="M-7.8 0Q-7.8 -4.8 -3 -4.8L3 -4.8Q7.8 -4.8 7.8 0L7.8 0Q7.8 4.8 3 4.8L-3 4.8Q-7.8 4.8 -7.8 0ZM-5 0Q-5 -2 -3 -2L3 -2Q5 -2 5 0L5 0Q5 2 3 2L-3 2Q-5 2 -5 0Z" fill-rule="evenodd" /></g>
        <g transform="translate(19.7 12.5)"><g filter="url(#sh-i1)"><g filter="url(#hi-i1)"><path d="M-7.8 0Q-7.8 -4.8 -3 -4.8L3 -4.8Q7.8 -4.8 7.8 0L7.8 0Q7.8 4.8 3 4.8L-3 4.8Q-7.8 4.8 -7.8 0ZM-5 0Q-5 -2 -3 -2L3 -2Q5 -2 5 0L5 0Q5 2 3 2L-3 2Q-5 2 -5 0Z" fill-rule="evenodd" fill="url(#upS-i1)" /></g></g><path stroke-width="2" fill="none" stroke="#5D7198" mask="url(#ol-i1-1)" d="M-7.8 0Q-7.8 -4.8 -3 -4.8L3 -4.8Q7.8 -4.8 7.8 0L7.8 0Q7.8 4.8 3 4.8L-3 4.8Q-7.8 4.8 -7.8 0ZM-5 0Q-5 -2 -3 -2L3 -2Q5 -2 5 0L5 0Q5 2 3 2L-3 2Q-5 2 -5 0Z" fill-rule="evenodd" /></g>
      </g></svg>`))
