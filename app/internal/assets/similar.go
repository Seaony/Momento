package assets

import (
	"context"
	"encoding/binary"
	"errors"
	"image"
	"io"
	"log/slog"
	"math/bits"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Seaony/Momento/app/internal/model"
	"golang.org/x/image/draw"
)

// SimilarDistance 是两张图算「相似」的最大差异位数（64 位差异哈希）。实测同图不同尺寸、轻度压缩在 0～4，
// 构图相近的不同照片一般在 10 以上；取 5 偏保守，宁可漏掉也不把不同的图并掉。
const SimilarDistance = 5

// DHash 是图的差异哈希：缩成 9×8 灰度，每行相邻两个像素比较亮度，得到 64 位。和尺寸、轻度压缩、调色无关。
func DHash(img image.Image) uint64 {
	small := image.NewGray(image.Rect(0, 0, 9, 8))
	draw.ApproxBiLinear.Scale(small, small.Bounds(), img, img.Bounds(), draw.Src, nil)
	var hash uint64
	for y := range 8 {
		for x := range 8 {
			hash <<= 1
			if small.GrayAt(x, y).Y < small.GrayAt(x+1, y).Y {
				hash |= 1
			}
		}
	}
	return hash
}

// SimilarColor 是两张图平均颜色每个通道最多差多少还算同一张。差异哈希只看明暗走向，
// 纯色色卡、同一渐变换了颜色的图哈希都一样，靠平均颜色分开。
const SimilarColor = 24

// Fingerprint 是判断两张图是不是同一张用的：差异哈希加平均颜色。
type Fingerprint struct {
	Hash  uint64
	Color [3]uint8
}

// ImageFingerprint 把图缩成 9×8 一次，从里面算差异哈希和平均颜色。
func ImageFingerprint(img image.Image) Fingerprint {
	small := image.NewRGBA(image.Rect(0, 0, 9, 8))
	draw.ApproxBiLinear.Scale(small, small.Bounds(), img, img.Bounds(), draw.Src, nil)
	var sum [3]int
	for i := 0; i < len(small.Pix); i += 4 {
		for c := range 3 {
			sum[c] += int(small.Pix[i+c])
		}
	}
	n := len(small.Pix) / 4
	return Fingerprint{Hash: DHash(small), Color: [3]uint8{uint8(sum[0] / n), uint8(sum[1] / n), uint8(sum[2] / n)}}
}

// Fingerprints 读每个素材的缩略图算指纹（缩略图最长边 512，比读原图快得多），多核并行。
// 只比静态图：视频、实况照片、GIF、APNG、动态 WebP 只看封面会把内容不同的当成一张，合并后另一张的动画或视频会跟着进回收站；
// 用通用格式封面的（这台机器解不开的 RAW、PSD 等）封面都一样，也不比。这些都不在返回结果里。
// 缩略图缺了或解不开的跳过并记日志，不让整次查找失败；取消时返回 ctx 的错误。
// progress 在工作协程里串行调用，调用方自己转到界面线程。
func (s *Service) Fingerprints(ctx context.Context, items []model.Asset, progress func(done, total int)) (map[string]Fingerprint, error) {
	placeholders, err := s.placeholderPosters(ctx)
	if err != nil {
		return nil, err
	}
	hashes := make(map[string]Fingerprint, len(items))
	var mu, progressMu sync.Mutex
	var done, skipped, excluded atomic.Int64
	reported := 0
	jobs := make(chan model.Asset)
	var wg sync.WaitGroup
	for range max(1, runtime.NumCPU()) {
		wg.Go(func() {
			for asset := range jobs {
				if still, err := s.stillImage(asset, placeholders); err != nil {
					skipped.Add(1)
					slog.Warn("算相似度跳过素材", "id", asset.ID, "error", err)
				} else if !still {
					excluded.Add(1)
				} else if hash, err := s.thumbnailFingerprint(asset.ID); err != nil {
					skipped.Add(1)
					slog.Warn("算相似度跳过素材", "id", asset.ID, "error", err)
				} else {
					mu.Lock()
					hashes[asset.ID] = hash
					mu.Unlock()
				}
				// 十万张不能每张都通知界面：每 64 张或做完时报一次，加锁保证报出去的数只增不减。
				if n := int(done.Add(1)); progress != nil && (n%64 == 0 || n == len(items)) {
					progressMu.Lock()
					if n > reported {
						reported = n
						progress(n, len(items))
					}
					progressMu.Unlock()
				}
			}
		})
	}
	for _, asset := range items {
		select {
		case jobs <- asset:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		}
	}
	close(jobs)
	wg.Wait()
	slog.Info("算相似度完成", "total", len(items), "fingerprinted", len(hashes), "notStill", excluded.Load(), "skipped", skipped.Load())
	return hashes, ctx.Err()
}

// placeholderPosters 是缩略图用了通用格式封面的素材。
func (s *Service) placeholderPosters(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM assets WHERE json_extract(exif_json,'$.posterUnavailable')='1'")
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Join(err, rows.Close())
		}
		ids[id] = true
	}
	return ids, errors.Join(rows.Err(), rows.Close())
}

// stillImage 判断素材是不是有真实缩略图的静态图。PNG、WebP 要读文件头才知道是不是动画。
func (s *Service) stillImage(asset model.Asset, placeholders map[string]bool) (bool, error) {
	if asset.Kind != model.KindImage && asset.Kind != "" || placeholders[asset.ID] {
		return false, nil
	}
	switch strings.ToLower(asset.Extension) {
	case "gif", "apng":
		return false, nil
	case "png", "webp":
		path, err := s.assetPath(asset.RelativePath)
		if err != nil {
			return false, err
		}
		f, err := os.Open(path)
		if err != nil {
			return false, err
		}
		defer f.Close()
		animated, err := IsAnimatedImage(f)
		return !animated, err
	}
	return true, nil
}

// IsAnimatedImage 看 PNG 在图像数据前有没有动画控制块（acTL，就是 APNG），或 WebP 扩展头里有没有动画标记。
// 只读文件头和 PNG 的块头，跳过块内容，不读整张图。其他格式返回 false。
func IsAnimatedImage(r io.ReadSeeker) (bool, error) {
	head := make([]byte, 21)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false, err
	}
	head = head[:n]
	switch {
	case len(head) >= 21 && string(head[:4]) == "RIFF" && string(head[8:16]) == "WEBPVP8X":
		return head[20]&0x02 != 0, nil
	case len(head) >= 8 && string(head[:8]) == "\x89PNG\r\n\x1a\n":
		at := int64(8)
		chunk := make([]byte, 8)
		for {
			if _, err := r.Seek(at, io.SeekStart); err != nil {
				return false, err
			}
			if _, err := io.ReadFull(r, chunk); err != nil {
				// 文件截断了还没碰到图像数据，当成不是动画，交给解码去报错。
				return false, nil
			}
			switch string(chunk[4:8]) {
			case "acTL":
				return true, nil
			case "IDAT", "IEND":
				return false, nil
			}
			at += 12 + int64(binary.BigEndian.Uint32(chunk[:4]))
		}
	}
	return false, nil
}

func (s *Service) thumbnailFingerprint(id string) (Fingerprint, error) {
	path, err := ThumbnailPath(s.root, id)
	if err != nil {
		return Fingerprint{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Fingerprint{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return Fingerprint{}, err
	}
	return ImageFingerprint(img), nil
}

// SimilarGroups 把相似的素材归成组，只留两张以上的组。相似是指差异哈希相近、平均颜色和长宽比也相近。
// 组里任意两张都相似，不是一串连起来的：否则黑、深灰、浅灰、白能一张挨一张连成一组，保留黑的会把白的也并掉，
// 用户在组里改选保留哪张也一样成立。
// 每组按 BestFirst 排好，第一张是建议保留的；组之间按最大那张的添加时间从新到旧。
func SimilarGroups(items []model.Asset, prints map[string]Fingerprint) [][]model.Asset {
	candidates, alike, components := similarComponents(items, prints)
	var groups [][]model.Asset
	for _, members := range components {
		slices.SortStableFunc(members, func(i, j int) int { return BestFirst(candidates[i], candidates[j]) })
		// 从最好的那张开始，把和组里每张都相似的收进来；没收进的留给下一轮另起一组。
		for len(members) > 1 {
			clique, rest := members[:1:1], []int{}
			for _, i := range members[1:] {
				if !slices.ContainsFunc(clique, func(j int) bool { return !alike(i, j) }) {
					clique = append(clique, i)
				} else {
					rest = append(rest, i)
				}
			}
			if len(clique) > 1 {
				group := make([]model.Asset, len(clique))
				for k, i := range clique {
					group[k] = candidates[i]
				}
				groups = append(groups, group)
			}
			members = rest
		}
	}
	slices.SortFunc(groups, func(a, b []model.Asset) int {
		if c := b[0].CreatedAt.Compare(a[0].CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a[0].ID, b[0].ID)
	})
	return groups
}

// similarComponents 用并查集把两两相似的连起来，返回有指纹的素材、两张是否相似的判断，以及两张以上的连通块（素材下标）。
func similarComponents(items []model.Asset, prints map[string]Fingerprint) ([]model.Asset, func(i, j int) bool, [][]int) {
	candidates := make([]model.Asset, 0, len(items))
	for _, asset := range items {
		if _, ok := prints[asset.ID]; ok {
			candidates = append(candidates, asset)
		}
	}
	values := make([]uint64, len(candidates))
	colors := make([][3]uint8, len(candidates))
	for i, asset := range candidates {
		values[i], colors[i] = prints[asset.ID].Hash, prints[asset.ID].Color
	}
	alike := func(i, j int) bool {
		return bits.OnesCount64(values[i]^values[j]) <= SimilarDistance && similarColor(colors[i], colors[j]) && similarShape(candidates[i], candidates[j])
	}
	parent := make([]int, len(candidates))
	for i := range parent {
		parent[i] = i
	}
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	// 两两比较十万张要一百多秒。差异不超过 5 位的两个哈希切成 6 段后，至少有一段完全相同，
	// 所以按每一段的值分桶，只比同一个桶里的。
	for _, chunk := range similarChunks {
		buckets := map[uint64][]int{}
		for i, value := range values {
			key := value >> chunk.shift & (1<<chunk.bits - 1)
			buckets[key] = append(buckets[key], i)
		}
		for _, members := range buckets {
			for x, i := range members {
				for _, j := range members[x+1:] {
					if find(i) != find(j) && alike(i, j) {
						parent[find(i)] = find(j)
					}
				}
			}
		}
	}
	byRoot := map[int][]int{}
	for i := range candidates {
		byRoot[find(i)] = append(byRoot[find(i)], i)
	}
	var components [][]int
	for _, members := range byRoot {
		if len(members) > 1 {
			components = append(components, members)
		}
	}
	return candidates, alike, components
}

// similarChunks 把 64 位切成 SimilarDistance+1 段。
var similarChunks = []struct{ shift, bits uint }{{0, 11}, {11, 11}, {22, 11}, {33, 11}, {44, 10}, {54, 10}}

func similarColor(a, b [3]uint8) bool {
	for c := range 3 {
		if max(a[c], b[c])-min(a[c], b[c]) > SimilarColor {
			return false
		}
	}
	return true
}

// 长宽比差 10% 以上的不算同一张（裁过的图、横竖不同的图）。尺寸不知道的不拦。
func similarShape(a, b model.Asset) bool {
	if a.Width <= 0 || a.Height <= 0 || b.Width <= 0 || b.Height <= 0 {
		return true
	}
	ra, rb := float64(a.Width)/float64(a.Height), float64(b.Width)/float64(b.Height)
	return ra/rb < 1.1 && rb/ra < 1.1
}

// BestFirst 决定一组相似图里保留哪张：像素多的、文件大的、先添加的在前。
func BestFirst(a, b model.Asset) int {
	if pa, pb := a.Width*a.Height, b.Width*b.Height; pa != pb {
		return pb - pa
	}
	if a.Size != b.Size {
		if a.Size > b.Size {
			return -1
		}
		return 1
	}
	if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}
