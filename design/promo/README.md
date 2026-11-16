# README 宣传图

- `content/01-content-squares.png`、`content/02-content-portraits.png`：Codex 生成的素材拼图（提示词在 `prompts/01`、`prompts/02`），切成 10 张图当宣传截图里的图库内容，不用别人的作品。
- 切图：方形拼图按 512 像素一格、竖版拼图按 512×768 一格切开，每边内缩 4 像素，放大 4 倍存成 JPEG，放到 `content/tiles/`（不进仓库），文件名就是界面上显示的图片名。
- 主界面截图：在 `app/` 里执行 `MOMENTO_PROMO_IMAGES=<tiles 目录> MOMENTO_PROMO_OUT=<输出目录> go test ./ui -run '^TestPromoScreenshots$' -count=1`，按 2 倍渲染中英文、浅深色四张（不进仓库）。
- 主图：把浅色截图和应用图标交给 Codex，按 `prompts/11-hero-en.md`、`prompts/12-hero-zh.md` 合成，成品在 `.github/assets/momento-hero-*.png`。
