# 图片格式测试样例

这八张图片由本仓库的 `generate.go` 生成渐变色块后编码，没有使用用户图片或第三方图片，可随仓库再分发。图片尺寸均为 96 × 64。

在 `app/` 中重新生成 JPEG、PNG、GIF、BMP 和 TIFF：

```sh
go run ./internal/assets/testdata/generate.go
```

再从生成的 PNG 转换其他格式：

```sh
cwebp -quiet -lossless internal/assets/testdata/sample.png -o internal/assets/testdata/sample.webp
sips -s format heic internal/assets/testdata/sample.png --out internal/assets/testdata/sample.heic
sips -s format avif internal/assets/testdata/sample.png --out internal/assets/testdata/sample.avif
```

`cwebp` 来自 Google WebP 编码工具，`sips` 来自 macOS，仅用于重新制作样例。应用和测试解码使用 Go 依赖，不调用这些工具，也不使用 cgo。

| 文件 | 编码方式 |
| --- | --- |
| sample.jpg | Go 标准库 JPEG，质量 90 |
| sample.png | Go 标准库 PNG |
| sample.gif | Go 标准库 GIF |
| sample.bmp | golang.org/x/image/bmp |
| sample.tiff | golang.org/x/image/tiff，Deflate |
| sample.webp | cwebp，无损 |
| sample.heic | macOS sips HEIC |
| sample.avif | macOS sips AVIF |
