// Copyright (c) 2026 Kubai087. Licensed under the Kubai Icons License; see LICENSE.
// Momento 的静态物件图标；由原版引擎负责渐变、外描边与导出。
const OL = 'class="ol" data-t="stroke:#5393FF"';
const DEEP = 'data-t="fill:#2F6FE0"';
const DEEPS = 'data-t="stroke:#2F6FE0"';
const CREAM = '#FFF6EC';
const face = (d) => `<path d="${d}" fill="${CREAM}"/>`;
const mark = (d, width = 1.6, extra = '') => `<path d="${d}" fill="none" ${DEEPS} stroke-width="${width}" stroke-linecap="round" stroke-linejoin="round" ${extra}/>`;
const dot = (x, y, r = 1.5) => `<circle cx="${x}" cy="${y}" r="${r}" ${DEEP}/>`;
const piece = (u, d, front = false, shadow = false) => `${shadow ? `<g filter="${u.sh}">` : ''}<g filter="${u.hi}"><path d="${d}" fill="${front ? u.upS : u.loS}"/></g>${shadow ? '</g>' : ''}<path d="${d}" ${OL}/>`;
const badge = (u, content) => piece(u, circ(15.5, 12.5, 10.8), false, true) + face(circ(15.5, 12.5, 7.6)) + content;
const windowBody = (u) => piece(u, rr(4, 3.3, 23, 18.4, 3), false, true) + face(rr(6.8, 8.6, 17.4, 10.4, 1.5));
const folderBody = (u) => {
  const back = rp([[4, 4], [11.3, 4], [14.3, 7], [27, 7], [27, 22], [4, 22]], [3, 1.6, 1.6, 3, 3, 3]);
  return piece(u, back, false, true) + face(rr(7, 7.2, 16.8, 11.5, 1.5)) + piece(u, rr(4, 12, 23, 10, 2.8), true);
};
const tagShape = rp([[4.6, 12.5], [11.4, 4.4], [24.4, 4.4], [26.4, 6.4], [26.4, 18.6], [24.4, 20.6], [11.4, 20.6]], [1.7, 2, 2, 1.8, 1.8, 2, 2]);
const tagBody = (u) => piece(u, tagShape, false, true) + face(circ(11.3, 12.5, 1.9));
const arrow = (direction) => direction === 'left'
  ? mark('M20 12.5H11', 2.2) + `<path d="${rp([[14.4, 7.6], [8.4, 12.5], [14.4, 17.4]], .7)}" ${DEEP}/>`
  : mark('M11 12.5H20', 2.2) + `<path d="${rp([[16.6, 7.6], [22.6, 12.5], [16.6, 17.4]], .7)}" ${DEEP}/>`;
const tray = (u, up) => {
  const body = rp([[4, 11.5], [10, 11.5], [11.7, 15.5], [19.3, 15.5], [21, 11.5], [27, 11.5], [27, 22], [4, 22]], [2.5, 1.2, 1.2, 1.2, 1.2, 2.5, 2.8, 2.8]);
  return piece(u, body, false, true) + face(rr(7.2, 17.5, 16.6, 2.7, 1.2)) + (up
    ? mark('M15.5 13.4V6.2', 2.2) + `<path d="${rp([[10.7, 7.5], [15.5, 2], [20.3, 7.5]], .6)}" ${DEEP}/>`
    : mark('M15.5 2.4V9.5', 2.2) + `<path d="${rp([[10.7, 8], [15.5, 13.5], [20.3, 8]], .6)}" ${DEEP}/>`);
};
const eyeBody = (u, closed) => {
  const body = 'M3.8 12.5C7.1 6.9 11.1 4.7 15.5 4.7C19.9 4.7 23.9 6.9 27.2 12.5C23.9 18.1 19.9 20.3 15.5 20.3C11.1 20.3 7.1 18.1 3.8 12.5Z';
  return piece(u, body, false, true) + face(circ(15.5, 12.5, 5.4)) + dot(15.5, 12.5, 2.6) + (closed ? mark('M6 3.2L25 21.8', 2.2) : '');
};
const album = (u, plus) => piece(u, rr(5.6, 2.2, 19.8, 20.6, 3), false, true) + face(rr(9, 5.5, 13, 13.8, 1.5))
  + mark('M7.6 5.2V19.8') + (plus ? mark('M15.5 9.2V15.6M12.3 12.4H18.7', 2.2) : `<path d="${rp([[10.5, 17.4], [14, 12], [17.2, 15.5], [20.5, 10.4], [20.5, 17.4]], 1)}" data-t="fill:#89B5FF"/>`);
const ICONS = [
  { name: 'photos', svg: (u) => piece(u, rr(4.4, 2.8, 20, 17.5, 3), false, true) + piece(u, rr(6.6, 5.4, 20, 17.5, 3), true) + face(rr(9.2, 8, 14.8, 12.2, 1.5)) + `<path d="${rp([[9.2, 18.7], [13.7, 12.5], [17.4, 16.3], [20.1, 13.5], [24, 18.7], [24, 20.2], [9.2, 20.2]], 1)}" data-t="fill:#89B5FF"/>` + dot(20.2, 11.5, 1.4) },
  { name: 'recent', svg: (u) => piece(u, circ(15.5, 12.5, 11), false, true) + face(circ(15.5, 12.5, 7.9)) + mark('M15.5 12.5L10.8 9.3', 2.6, 'stroke-opacity=".5"') + mark('M15.5 12.5L19.8 7.5', 2) + dot(15.5, 12.5, 1.5) },
  { name: 'favorite', svg: (u) => {
    const heart = 'M15.5 22.4C12 20.2 4.7 14.7 4.7 8.7C4.7 2 12.3 1.2 15.5 6C18.7 1.2 26.3 2 26.3 8.7C26.3 14.7 19 20.2 15.5 22.4Z';
    return piece(u, heart, false, true) + face('M15.5 17.9C12.7 16.2 9.4 12.7 9.4 9.8C9.4 6.8 13.3 6.6 15.5 9.8C17.7 6.6 21.6 6.8 21.6 9.8C21.6 12.7 18.3 16.2 15.5 17.9Z');
  } },
  { name: 'random', svg: (u) => piece(u, rr(5.2, 2.2, 20.6, 20.6, 4.2), false, true) + face(rr(8.3, 5.3, 14.4, 14.4, 2.6)) + dot(11.8, 8.8, 1.5) + dot(15.5, 12.5, 1.5) + dot(19.2, 16.2, 1.5) },
  { name: 'folder', svg: folderBody },
  { name: 'smart-folder', svg: (u) => folderBody(u) + `<path d="${rp([[21.8, 10.2], [23.4, 13.9], [27, 15.5], [23.4, 17.1], [21.8, 20.8], [20.2, 17.1], [16.6, 15.5], [20.2, 13.9]], .6)}" fill="${CREAM}" stroke-width="1.2" ${DEEPS}/>` },
  { name: 'unfiled', svg: (u) => piece(u, rr(4, 4, 23, 17, 3), false, true) + face(rr(7, 6.7, 17, 7.4, 1.5)) + piece(u, rr(4, 12.4, 23, 9.6, 2.8), true) + mark('M12.3 16.8H18.7') },
  { name: 'tag', svg: (u) => tagBody(u) + mark('M17.2 9H22M17.2 15.5H22') },
  { name: 'untagged', svg: (u) => tagBody(u) + mark('M17.2 12.5H23', 1.6, 'stroke-dasharray="2.4 2.4"') },
  { name: 'tags', svg: (u) => piece(u, rr(5, 3.4, 19.6, 15.4, 2.8), false, true) + `<g transform="translate(1.4 2)">${piece(u, tagShape, true) + face(circ(11.3, 12.5, 1.9)) + mark('M17.2 12.5H22')}</g>` },
  { name: 'trash', svg: (u) => piece(u, rp([[7.1, 6.6], [23.9, 6.6], [22.3, 23], [8.7, 23]], [1.2, 1.2, 2.8, 2.8]), false, true) + face(rr(11.3, 9, 8.4, 10.6, 1.6)) + mark('M15.5 11.6V16.9') + piece(u, rr(11.2, 2, 8.6, 4, 1.4), false) + piece(u, rr(5.6, 4.8, 19.8, 3.5, 1.7), true) },
  { name: 'settings', svg: (u) => piece(u, gear(6, 11.2, 8.9), false, true) + face(circ(15.5, 12.5, 5.4)) + piece(u, circ(15.5, 12.5, 2.4), true) },
  { name: 'link', svg: (u) => `<g transform="rotate(-35 15.5 12.5)">${piece(u, rr(4.2, 8.1, 14.4, 8.8, 4.4), false, true) + face(rr(7, 10.9, 8.8, 3.2, 1.6)) + piece(u, rr(12.4, 8.1, 14.4, 8.8, 4.4), true) + face(rr(15.2, 10.9, 8.8, 3.2, 1.6)) + mark('M12.2 12.5H19')}</g>` },
  { name: 'external-link', svg: (u) => piece(u, rr(4.8, 5.2, 19.8, 17.6, 3), false, true) + face(rr(7.8, 8.2, 13.8, 11.6, 1.5)) + mark('M14 16L24.5 5.5', 2.2) + `<path d="${rp([[18.5, 2.9], [27, 2.9], [27, 11.4]], .8)}" ${DEEP}/>` },
  { name: 'thumbnail-size', svg: (u) => piece(u, rr(4, 4, 23, 17, 3), false, true) + face(rr(7, 7, 8, 10.6, 1.4)) + face(rr(18, 11, 5.8, 6.6, 1.3)) },
  { name: 'filter', svg: (u) => {
    const funnel = rp([[4.8, 3.2], [26.2, 3.2], [18.3, 12.6], [18.3, 21.4], [12.7, 23], [12.7, 12.6]], [2, 2, 1.5, 1.5, 1.5, 1.5]);
    return piece(u, funnel, false, true) + face(rr(8.9, 5.9, 13.2, 3.3, 1.5)) + piece(u, rr(12.7, 14.2, 5.6, 6.8, 1.2), true);
  } },
  { name: 'sidebar', svg: (u) => windowBody(u) + piece(u, rr(5.6, 8.6, 5.6, 11.6, 1.5), true) },
  { name: 'inspector', svg: (u) => windowBody(u) + piece(u, rr(19.8, 8.6, 5.6, 11.6, 1.5), true) },
  { name: 'import', svg: (u) => tray(u, false) },
  { name: 'export', svg: (u) => tray(u, true) },
  { name: 'sort', svg: (u) => piece(u, rr(4.4, 2.8, 15.7, 19.4, 2.8), false, true) + face(rr(7.2, 5.6, 10.1, 13.8, 1.4)) + mark('M10 8.8H14.7M10 13.4H12.6') + mark('M24.3 4.3V15.8', 2.2) + `<path d="${rp([[20.7, 15.3], [24.3, 20.1], [27.9, 15.3]], .6)}" ${DEEP}/>` },
  { name: 'masonry', svg: (u) => piece(u, rr(4, 3.3, 23, 18.4, 3), false, true) + face(rr(6.9, 6.5, 7.4, 4.5, 1.4)) + face(rr(16.8, 6.5, 7.4, 7.5, 1.4)) + face(rr(6.9, 13.4, 7.4, 5.4, 1.3)) + face(rr(16.8, 16.4, 7.4, 2.7, 1.2)) },
  { name: 'grid', svg: (u) => piece(u, rr(4.6, 2.2, 21.8, 20.6, 3), false, true) + [[7.4, 5.2], [17.2, 5.2], [7.4, 14], [17.2, 14]].map(([x,y]) => face(rr(x,y,6.4,5.8,1.4))).join('') },
  { name: 'list', svg: (u) => piece(u, rr(4, 3.3, 23, 18.4, 3), false, true) + [6.2,11.3,16.4].map((y) => face(rr(7,y,17,2.6,1.2))).join('') },
  { name: 'search', svg: (u) => piece(u, rp([[20.4, 15.2], [27.3, 21.2], [24.8, 23.9], [18, 17.7]], 1.6), true, true) + piece(u, circ(12.8, 9.6, 8.4), false, true) + face(circ(12.8, 9.6, 5.5)) },
  { name: 'back', svg: (u) => badge(u, arrow('left')) },
  { name: 'forward', svg: (u) => badge(u, arrow('right')) },
  { name: 'down', svg: (u) => `<g transform="rotate(90 15.5 12.5)">${badge(u, arrow('right'))}</g>` },
  { name: 'plus', svg: (u) => badge(u, mark('M15.5 7.4V17.6M10.4 12.5H20.6', 2.2)) },
  { name: 'minus', svg: (u) => badge(u, mark('M10.4 12.5H20.6', 2.2)) },
  { name: 'close', svg: (u) => badge(u, mark('M11.6 8.6L19.4 16.4M19.4 8.6L11.6 16.4', 2.2)) },
  { name: 'check', svg: (u) => badge(u, mark('M10 12.6L13.6 16L21.1 8.7', 2.2)) },
  { name: 'chevrons-up-down', svg: (u) => piece(u, rr(7.4, 1.8, 16.2, 21.4, 4.2), false, true) + face(rr(10.2, 4.6, 10.6, 15.8, 2.4)) + `<path d="${rp([[11.7, 9.8], [15.5, 5.7], [19.3, 9.8]], .6)}" ${DEEP}/><path d="${rp([[11.7, 15.2], [15.5, 19.3], [19.3, 15.2]], .6)}" ${DEEP}/>` },
  { name: 'play', svg: (u) => badge(u, `<path d="${rp([[12, 7.1], [21, 12.5], [12, 17.9]], .9)}" ${DEEP}/>` ) },
  { name: 'pause', svg: (u) => badge(u, `<path d="${rr(10.8, 7.4, 3.3, 10.2, 1.1)}${rr(16.9, 7.4, 3.3, 10.2, 1.1)}" ${DEEP}/>` ) },
  { name: 'lock', svg: (u) => piece(u, rr(9.5, 1.8, 12, 17.2, 5.6), false, true) + face(rr(12.5, 4.8, 6, 10, 2.8)) + piece(u, rr(5.6, 10.2, 19.8, 12.8, 3), true) + dot(15.5, 15, 2) + mark('M15.5 15.8V19.3', 2.2) },
  { name: 'controls', svg: (u) => piece(u, rr(7.6, 1.4, 15.8, 22.2, 7.9), false, true) + face(rp([[10.4, 10.4], [20.6, 10.4], [20.6, 15.6], [20.6, 20.2], [10.4, 20.2]], [1.5,1.5,4.5,4.5,4.5])) + mark('M15.5 5.1V8.3', 2.2) },
  { name: 'keyboard', svg: (u) => piece(u, rr(4, 4.2, 23, 16.6, 3), false, true) + [[7.2,7.2],[13.9,7.2],[20.6,7.2],[7.2,12.4],[20.6,12.4]].map(([x,y]) => face(rr(x,y,3.4,2.7,.9))).join('') + face(rr(12.5,12.4,5.2,5.2,1.1)) + face(rr(7.2,16.9,3.4,2.4,.8)) + face(rr(20.6,16.9,3.4,2.4,.8)) },
  { name: 'notifications', svg: (u) => dot(15.5, 3.1, 1.9) + dot(15.5, 21.4, 2.3) + piece(u, 'M6.5 19C8 16.8 8.4 15 8.4 11.3C8.4 7 11.4 4.5 15.5 4.5C19.6 4.5 22.6 7 22.6 11.3C22.6 15 23 16.8 24.5 19Z', false, true) + face(rr(12.2,7.6,6.6,7.9,2.6)) + piece(u, rr(5.6,17.5,19.8,3.2,1.6), true) },
  { name: 'appearance', svg: (u) => {
    const body = 'M26.7 10.5C26.7 4.7 20.6 1.4 15.2 1.6C8.8 1.8 4.1 6.5 4.1 12.6C4.1 18.6 8.8 23.3 14.6 23.3C18 23.3 19 21.4 17.5 19.6C15.9 17.7 17 16 20.1 16.3C24.8 16.6 26.7 14.5 26.7 10.5Z';
    return piece(u, body, false, true) + face(circ(10,10.3,2)) + face(circ(16.2,6.5,2)) + face(circ(22.2,10.1,2)) + face(circ(10.5,17,2));
  } },
  { name: 'browser', svg: (u) => windowBody(u) + face(circ(15.5, 12.8, 6.2)) + `<ellipse cx="15.5" cy="12.8" rx="2.8" ry="6.2" fill="none" ${DEEPS} stroke-width="1.6"/>` + mark('M9.3 12.8H21.7') },
  { name: 'info', svg: (u) => badge(u, dot(15.5,7.8,1.5) + mark('M15.5 12.1V17.5',2.2)) },
  { name: 'eye', svg: (u) => eyeBody(u, false) },
  { name: 'eye-off', svg: (u) => eyeBody(u, true) },
  { name: 'edit', svg: (u) => `<g transform="rotate(-45 15.5 12.5)">${piece(u, rr(5.2,8.5,18.4,8,2), false, true) + face(rr(8.5,10.7,11,3.6,1.2)) + piece(u, rp([[23.6,8.5],[29,12.5],[23.6,16.5]], [0,1,0]), true) + mark('M5.2 8.5V16.5')}</g>` },
  { name: 'library', svg: (u) => album(u, false) },
  { name: 'create-library', svg: (u) => album(u, true) },
  { name: 'extension', svg: (u) => {
    const body = 'M6.2 5.4H12V3.8C12 0.2 19 0.2 19 3.8V5.4H24.8Q27 5.4 27 7.6V11H25.5C22 11 22 17.5 25.5 17.5H27V20.8Q27 23 24.8 23H19V21.5C19 18 12 18 12 21.5V23H6.2Q4 23 4 20.8V7.6Q4 5.4 6.2 5.4Z';
    return piece(u, body, false, true) + face(rr(8,9.3,10.4,6.4,1.8));
  } },
  { name: 'warning', svg: (u) => piece(u, rp([[15.5,1.4],[28,22.2],[3,22.2]],2.2), false, true) + face(rp([[15.5,6.2],[23.6,19.7],[7.4,19.7]],1.2)) + mark('M15.5 9.5V13.6',2.2) + dot(15.5,17,1.4) },
  { name: 'rotate-right', svg: (u) => badge(u, mark('M20 15.2A5.2 5.2 0 1 1 18.5 8.2',2.2) + `<path d="${rp([[20.3,5.2],[20.3,11.2],[14.3,11.2]],.6)}" ${DEEP}/>` ) },
  { name: 'rotate-left', svg: (u) => `<g transform="translate(31 0) scale(-1 1)">${badge(u, mark('M20 15.2A5.2 5.2 0 1 1 18.5 8.2',2.2) + `<path d="${rp([[20.3,5.2],[20.3,11.2],[14.3,11.2]],.6)}" ${DEEP}/>` )}</g>` },
];
for (const icon of ICONS) { icon.target = () => ({}); icon.draw = () => {}; }
