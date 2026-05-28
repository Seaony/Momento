// 中文注释：本服务从图片缩略采样中提取主色，用于检查器色板和颜色筛选。
import CoreGraphics
import Foundation
import ImageIO

struct AssetColorAnalysisService: Sendable {
    private let sampleMaxPixelSize: Int
    private let mergeDistance: Double

    init(sampleMaxPixelSize: Int = 128, mergeDistance: Double = 28) {
        self.sampleMaxPixelSize = sampleMaxPixelSize
        self.mergeDistance = mergeDistance
    }

    nonisolated func paletteColors(
        for url: URL,
        libraryID: String,
        assetID: String,
        maxColorCount: Int = 8
    ) -> [AssetColor] {
        guard maxColorCount > 0,
              let image = sampleImage(for: url),
              let pixels = rgbaPixels(from: image) else {
            return []
        }

        var buckets: [Int: ColorBucket] = [:]
        var consideredPixelCount = 0

        for pixel in pixels where pixel.alpha >= 13 {
            consideredPixelCount += 1
            let key = quantizedKey(red: pixel.red, green: pixel.green, blue: pixel.blue)
            buckets[key, default: ColorBucket()].append(pixel)
        }

        guard consideredPixelCount > 0 else {
            return []
        }

        let merged = buckets.values
            .sorted { lhs, rhs in
                if lhs.count == rhs.count {
                    return lhs.hex < rhs.hex
                }
                return lhs.count > rhs.count
            }
            .reduce(into: [ColorBucket]()) { partialResult, bucket in
                if let index = partialResult.firstIndex(where: { $0.distance(to: bucket) <= mergeDistance }) {
                    partialResult[index].merge(bucket)
                } else {
                    partialResult.append(bucket)
                }
            }
            .sorted { lhs, rhs in
                if lhs.count == rhs.count {
                    return lhs.hex < rhs.hex
                }
                return lhs.count > rhs.count
            }
            .prefix(maxColorCount)

        return merged.enumerated().map { index, bucket in
            AssetColor(
                id: "\(assetID)-color-\(index)",
                libraryID: libraryID,
                assetID: assetID,
                hex: bucket.hex,
                coverage: Double(bucket.count) / Double(consideredPixelCount),
                sortIndex: index
            )
        }
    }

    private nonisolated func sampleImage(for url: URL) -> CGImage? {
        guard let source = CGImageSourceCreateWithURL(url as CFURL, nil) else {
            return nil
        }

        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: sampleMaxPixelSize
        ]
        return CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary)
    }

    private nonisolated func rgbaPixels(from image: CGImage) -> [Pixel]? {
        let width = image.width
        let height = image.height
        guard width > 0, height > 0 else {
            return nil
        }

        let bytesPerPixel = 4
        let bytesPerRow = width * bytesPerPixel
        var data = [UInt8](repeating: 0, count: height * bytesPerRow)
        guard let colorSpace = CGColorSpace(name: CGColorSpace.sRGB),
              let context = CGContext(
                data: &data,
                width: width,
                height: height,
                bitsPerComponent: 8,
                bytesPerRow: bytesPerRow,
                space: colorSpace,
                bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
              ) else {
            return nil
        }

        context.draw(image, in: CGRect(x: 0, y: 0, width: width, height: height))

        return stride(from: 0, to: data.count, by: bytesPerPixel).map { offset in
            Pixel(
                red: data[offset],
                green: data[offset + 1],
                blue: data[offset + 2],
                alpha: data[offset + 3]
            )
        }
    }

    private nonisolated func quantizedKey(red: UInt8, green: UInt8, blue: UInt8) -> Int {
        let r = Int(red) / 16
        let g = Int(green) / 16
        let b = Int(blue) / 16
        return (r << 8) | (g << 4) | b
    }
}

nonisolated private struct Pixel: Sendable {
    var red: UInt8
    var green: UInt8
    var blue: UInt8
    var alpha: UInt8
}

nonisolated private struct ColorBucket: Sendable {
    var redTotal = 0
    var greenTotal = 0
    var blueTotal = 0
    var count = 0

    var red: Int { count == 0 ? 0 : redTotal / count }
    var green: Int { count == 0 ? 0 : greenTotal / count }
    var blue: Int { count == 0 ? 0 : blueTotal / count }

    var hex: String {
        String(format: "#%02X%02X%02X", red, green, blue)
    }

    mutating func append(_ pixel: Pixel) {
        redTotal += Int(pixel.red)
        greenTotal += Int(pixel.green)
        blueTotal += Int(pixel.blue)
        count += 1
    }

    mutating func merge(_ other: ColorBucket) {
        redTotal += other.redTotal
        greenTotal += other.greenTotal
        blueTotal += other.blueTotal
        count += other.count
    }

    func distance(to other: ColorBucket) -> Double {
        let redDelta = Double(red - other.red)
        let greenDelta = Double(green - other.green)
        let blueDelta = Double(blue - other.blue)
        return (redDelta * redDelta + greenDelta * greenDelta + blueDelta * blueDelta).squareRoot()
    }
}
