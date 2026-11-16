package assets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"math/bits"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"golang.org/x/image/draw"
)

func gradientImage(w, h int, flip bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			v := uint8((x*255/w + y*80/h) % 256)
			if (x/(w/5))%2 == 0 {
				v = 255 - v/2
			}
			if flip {
				v = 255 - v
			}
			img.Set(x, y, color.RGBA{v, v / 2, 255 - v, 255})
		}
	}
	return img
}

// 同一张图缩小、再压成低质量 JPEG，哈希几乎不变；内容不同的图差得远。
func TestDHashSimilarity(t *testing.T) {
	original := gradientImage(800, 600, false)
	small := image.NewRGBA(image.Rect(0, 0, 200, 150))
	draw.CatmullRom.Scale(small, small.Bounds(), original, original.Bounds(), draw.Src, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, small, &jpeg.Options{Quality: 40}); err != nil {
		t.Fatal(err)
	}
	compressed, err := jpeg.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	base := DHash(original)
	if d := bits.OnesCount64(base ^ DHash(compressed)); d > SimilarDistance {
		t.Fatalf("同一张图缩小压缩后差 %d 位", d)
	}
	if !similarColor(ImageFingerprint(original).Color, ImageFingerprint(compressed).Color) {
		t.Fatalf("同一张图缩小压缩后平均颜色差太多: %v %v", ImageFingerprint(original).Color, ImageFingerprint(compressed).Color)
	}
	if d := bits.OnesCount64(base ^ DHash(gradientImage(800, 600, true))); d <= SimilarDistance {
		t.Fatalf("不同的图只差 %d 位", d)
	}
}

func TestSimilarGroups(t *testing.T) {
	now := time.Now()
	items := []model.Asset{
		{ID: "a-small", Width: 400, Height: 300, Size: 10, CreatedAt: now},
		{ID: "a-big", Width: 1600, Height: 1200, Size: 900, CreatedAt: now.Add(-time.Hour)},
		{ID: "a-crop", Width: 300, Height: 300, Size: 10, CreatedAt: now},
		{ID: "b", Width: 400, Height: 300, CreatedAt: now},
		{ID: "no-hash", Width: 400, Height: 300},
	}
	gray := [3]uint8{128, 128, 128}
	items = append(items, model.Asset{ID: "a-red", Width: 400, Height: 300})
	hashes := map[string]Fingerprint{"a-small": {0b1011, gray}, "a-big": {0b1111, gray}, "a-crop": {0b1011, gray}, "b": {^uint64(0), gray}, "a-red": {0b1011, [3]uint8{200, 40, 40}}}
	groups := SimilarGroups(items, hashes)
	if len(groups) != 1 || len(groups[0]) != 2 || groups[0][0].ID != "a-big" || groups[0][1].ID != "a-small" {
		t.Fatalf("分组不对：%+v", groups)
	}
}

// 分桶只是提速：连通块必须和两两比较完全一样，包括正好差 5 位、差 6 位的边界；最后的每组里任意两张都相似。
func TestSimilarGroupsMatchesPairwise(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	items := []model.Asset{}
	hashes := map[string]Fingerprint{}
	for base := range 300 {
		root := rng.Uint64()
		for copy := range 4 {
			value := root
			for _, bit := range rng.Perm(64)[:rng.IntN(7)] {
				value ^= 1 << bit
			}
			id := fmt.Sprintf("%03d-%d", base, copy)
			items = append(items, model.Asset{ID: id, Width: 100, Height: 100})
			hashes[id] = Fingerprint{Hash: value}
		}
	}
	candidates, alike, components := similarComponents(items, hashes)
	var got [][]string
	for _, members := range components {
		var ids []string
		for _, i := range members {
			ids = append(ids, candidates[i].ID)
		}
		got = append(got, ids)
	}
	parent := map[string]string{}
	var find func(string) string
	find = func(id string) string {
		if p, ok := parent[id]; ok && p != id {
			return find(p)
		}
		return id
	}
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			if bits.OnesCount64(hashes[items[i].ID].Hash^hashes[items[j].ID].Hash) <= SimilarDistance {
				parent[find(items[i].ID)] = find(items[j].ID)
			}
		}
	}
	byRoot := map[string][]string{}
	for _, item := range items {
		byRoot[find(item.ID)] = append(byRoot[find(item.ID)], item.ID)
	}
	var want [][]string
	for _, ids := range byRoot {
		if len(ids) > 1 {
			want = append(want, ids)
		}
	}
	if len(want) < 100 {
		t.Fatalf("测试数据里相似组太少: %d", len(want))
	}
	if !slices.EqualFunc(sortGroups(got), sortGroups(want), slices.Equal) {
		t.Fatalf("分桶分组和两两比较不一致: got %d 组, want %d 组", len(got), len(want))
	}
	index := map[string]int{}
	for i, asset := range candidates {
		index[asset.ID] = i
	}
	for _, group := range SimilarGroups(items, hashes) {
		for x := range group {
			for y := x + 1; y < len(group); y++ {
				if !alike(index[group[x].ID], index[group[y].ID]) {
					t.Fatalf("同一组里 %s 和 %s 不相似", group[x].ID, group[y].ID)
				}
			}
		}
	}
}

// 黑到白一级级差 24 的纯灰图，哈希都是 0、相邻的颜色都算相似；不能连成一组，否则保留黑的会把白的也并掉。
func TestSimilarGroupsNoChaining(t *testing.T) {
	var items []model.Asset
	prints := map[string]Fingerprint{}
	for level := 0; level <= 240; level += 24 {
		id := fmt.Sprintf("gray-%03d", level)
		items = append(items, model.Asset{ID: id, Width: 100, Height: 100})
		prints[id] = Fingerprint{Color: [3]uint8{uint8(level), uint8(level), uint8(level)}}
	}
	groups := SimilarGroups(items, prints)
	if len(groups) == 0 {
		t.Fatal("相邻的灰度应该成组")
	}
	for _, group := range groups {
		for _, a := range group {
			for _, b := range group {
				if !similarColor(prints[a.ID].Color, prints[b.ID].Color) {
					t.Fatalf("%s 和 %s 颜色差太多却在一组：%v", a.ID, b.ID, groupIDs(groups))
				}
			}
		}
	}
}

// 动态 PNG（有 acTL）、带动画标记的 WebP 是动图；普通 PNG、静态 WebP、截断的文件不是。
func TestIsAnimatedImage(t *testing.T) {
	png := func(chunks ...string) []byte {
		data := []byte("\x89PNG\r\n\x1a\n")
		for _, name := range chunks {
			body := []byte("0123")
			data = binary.BigEndian.AppendUint32(data, uint32(len(body)))
			data = append(data, name...)
			data = append(data, body...)
			data = append(data, 0, 0, 0, 0)
		}
		return data
	}
	webp := func(flags byte) []byte {
		data := []byte("RIFF\x00\x00\x00\x00WEBPVP8X\x0a\x00\x00\x00")
		return append(data, flags, 0, 0, 0)
	}
	for name, test := range map[string]struct {
		data []byte
		want bool
	}{
		"apng":           {png("IHDR", "acTL", "IDAT", "IEND"), true},
		"png":            {png("IHDR", "iCCP", "IDAT", "acTL", "IEND"), false},
		"truncated-png":  {png("IHDR")[:20], false},
		"animated-webp":  {webp(0x02 | 0x10), true},
		"still-webp-ext": {webp(0x10), false},
		"jpeg":           {[]byte("\xff\xd8\xff\xe0"), false},
	} {
		got, err := IsAnimatedImage(bytes.NewReader(test.data))
		if err != nil || got != test.want {
			t.Errorf("%s: got %v %v, want %v", name, got, err, test.want)
		}
	}
}

func groupIDs(groups [][]model.Asset) [][]string {
	out := make([][]string, len(groups))
	for i, group := range groups {
		for _, asset := range group {
			out[i] = append(out[i], asset.ID)
		}
	}
	return out
}

func sortGroups(groups [][]string) [][]string {
	for _, ids := range groups {
		slices.Sort(ids)
	}
	slices.SortFunc(groups, func(a, b []string) int { return slices.Compare(a, b) })
	return groups
}

// 不同颜色的纯色色卡差异哈希一样（都是 0），平均颜色要把它们分开；同色的不同尺寸仍算一张。
func TestFingerprintSeparatesSolidColors(t *testing.T) {
	solid := func(w, h int, c color.RGBA) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < len(img.Pix); i += 4 {
			copy(img.Pix[i:], []uint8{c.R, c.G, c.B, c.A})
		}
		return img
	}
	red, blue, redSmall := ImageFingerprint(solid(400, 300, color.RGBA{220, 30, 30, 255})), ImageFingerprint(solid(400, 300, color.RGBA{30, 30, 220, 255})), ImageFingerprint(solid(200, 150, color.RGBA{220, 30, 30, 255}))
	if red.Hash != blue.Hash {
		t.Fatal("测试前提不成立：纯色图哈希应相同")
	}
	items := []model.Asset{{ID: "red", Width: 400, Height: 300}, {ID: "blue", Width: 400, Height: 300}, {ID: "red-small", Width: 200, Height: 150}}
	groups := groupIDs(SimilarGroups(items, map[string]Fingerprint{"red": red, "blue": blue, "red-small": redSmall}))
	if len(groups) != 1 || !slices.Equal(groups[0], []string{"red", "red-small"}) {
		t.Fatalf("分组不对：%v", groups)
	}
}

// 真实导入：同一张图的大图和压缩小图从缩略图算出来归成一组，大图排第一；内容不同的不进组；缩略图丢了的跳过不报错。
// 同一张图存成 GIF、标成通用封面、当成实况照片的都不参与比较。
func TestFingerprintsFromImportedThumbnails(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	write := func(name string, img image.Image) string {
		path := filepath.Join(dir, name)
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if strings.HasSuffix(name, ".gif") {
			err = gif.Encode(f, img, nil)
		} else {
			err = jpeg.Encode(f, img, &jpeg.Options{Quality: 60})
		}
		if err != nil {
			t.Fatal(err)
		}
		return path
	}
	scaled := func(w, h int, img image.Image) image.Image {
		out := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(out, out.Bounds(), img, img.Bounds(), draw.Src, nil)
		return out
	}
	original := gradientImage(1200, 900, false)
	paths := []string{write("big.jpg", original), write("small.jpg", scaled(400, 300, original)), write("other.jpg", gradientImage(1200, 900, true)),
		write("lost.jpg", gradientImage(1000, 800, true)), write("anim.gif", scaled(600, 450, original)), write("poster.jpg", scaled(800, 600, original)), write("live.jpg", scaled(1000, 750, original))}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: paths}, nil); err != nil {
		t.Fatal(err)
	}
	snapshot, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]model.Asset{}
	for i, asset := range snapshot.Assets {
		if asset.Name == "live" {
			snapshot.Assets[i].Kind = model.KindLive
		}
		byName[asset.Name] = snapshot.Assets[i]
	}
	if _, err := lib.DB.Exec(`UPDATE assets SET exif_json='{"posterUnavailable":"1"}' WHERE id=?`, byName["poster"].ID); err != nil {
		t.Fatal(err)
	}
	thumb, err := ThumbnailPath(lib.Root, byName["lost"].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(thumb); err != nil {
		t.Fatal(err)
	}
	prints, err := s.Fingerprints(ctx, snapshot.Assets, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"lost", "anim", "poster", "live"} {
		if _, ok := prints[byName[name].ID]; ok {
			t.Fatalf("%s 不该参与比较", name)
		}
	}
	if len(prints) != 3 {
		t.Fatalf("应只有 3 张静态图有指纹：%d", len(prints))
	}
	groups := groupIDs(SimilarGroups(snapshot.Assets, prints))
	if len(groups) != 1 || !slices.Equal(groups[0], []string{byName["big"].ID, byName["small"].ID}) {
		t.Fatalf("分组不对：%v", groups)
	}
}
