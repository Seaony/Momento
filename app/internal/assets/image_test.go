package assets

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestImageOrientations(t *testing.T) {
	src := image.NewNRGBA(image.Rect(3, 4, 5, 7))
	for i := range 6 {
		src.SetNRGBA(3+i%2, 4+i/2, color.NRGBA{R: uint8(i + 1), A: 255})
	}
	cases := []struct {
		width  int
		pixels []uint8
	}{
		{2, []uint8{1, 2, 3, 4, 5, 6}}, {2, []uint8{2, 1, 4, 3, 6, 5}}, {2, []uint8{6, 5, 4, 3, 2, 1}}, {2, []uint8{5, 6, 3, 4, 1, 2}},
		{3, []uint8{1, 3, 5, 2, 4, 6}}, {3, []uint8{5, 3, 1, 6, 4, 2}}, {3, []uint8{6, 4, 2, 5, 3, 1}}, {3, []uint8{2, 4, 6, 1, 3, 5}},
	}
	for i, tc := range cases {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			got := orient(src, i+1)
			b := got.Bounds()
			if b.Dx() != tc.width || b.Dy() != len(tc.pixels)/tc.width {
				t.Fatalf("size %v", b)
			}
			for p, want := range tc.pixels {
				r, _, _, _ := got.At(b.Min.X+p%tc.width, b.Min.Y+p/tc.width).RGBA()
				if uint8(r>>8) != want {
					t.Fatalf("pixel %d: %d != %d", p, r>>8, want)
				}
			}
		})
	}
}

func TestImageThumbnailLimitsAndAtomicReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thumbnails", "one.png")
	for _, size := range []image.Point{{1200, 600}, {3, 7}} {
		if err := WriteThumbnail(path, image.NewNRGBA(image.Rect(0, 0, size.X, size.Y))); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if size.X == 1200 && (cfg.Width != 512 || cfg.Height != 256) {
			t.Fatalf("large thumbnail: %+v", cfg)
		}
		if size.X == 3 && (cfg.Width != 3 || cfg.Height != 7) {
			t.Fatalf("small thumbnail: %+v", cfg)
		}
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".thumbnail-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files %v, %v", matches, err)
	}
}

type imageTestEXIFEntry struct {
	id, kind uint16
	count    uint32
	data     []byte
}

func imageTestShort(v uint16) []byte {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	return b[:]
}
func imageTestLong(v uint32) []byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	return b[:]
}
func imageTestRational(a, b uint32) []byte { return append(imageTestLong(a), imageTestLong(b)...) }
func imageTestTIFF() []byte {
	root := []imageTestEXIFEntry{
		{0x010f, 2, 6, []byte("Canon\x00")}, {0x0110, 2, 5, []byte("Tiny\x00")}, {0x0112, 3, 1, imageTestShort(6)},
		{0x011a, 5, 1, imageTestRational(300, 1)}, {0x011b, 5, 1, imageTestRational(300, 1)}, {0x0128, 3, 1, imageTestShort(2)},
		{0x013b, 2, 7, []byte("Keeper\x00")}, {0x8769, 4, 1, nil},
	}
	exif := []imageTestEXIFEntry{
		{0x829a, 5, 1, imageTestRational(1, 125)}, {0x829d, 5, 1, imageTestRational(28, 10)}, {0x8822, 3, 1, imageTestShort(2)},
		{0x8827, 3, 1, imageTestShort(400)}, {0x9003, 2, 20, []byte("2024:01:02 03:04:05\x00")},
		{0x9207, 3, 1, imageTestShort(5)}, {0x9209, 3, 1, imageTestShort(0)}, {0x920a, 5, 1, imageTestRational(50, 1)},
		{0xa001, 3, 1, imageTestShort(1)}, {0xa403, 3, 1, imageTestShort(0)}, {0xa434, 2, 5, []byte("Lens\x00")},
	}
	exifOffset := 8 + 2 + 12*len(root) + 4
	dataOffset := exifOffset + 2 + 12*len(exif) + 4
	out := make([]byte, dataOffset)
	copy(out, []byte{'I', 'I', 42, 0, 8, 0, 0, 0})
	root[len(root)-1].data = imageTestLong(uint32(exifOffset))
	for _, dir := range []struct {
		offset  int
		entries []imageTestEXIFEntry
	}{{8, root}, {exifOffset, exif}} {
		binary.LittleEndian.PutUint16(out[dir.offset:], uint16(len(dir.entries)))
		for i, entry := range dir.entries {
			off := dir.offset + 2 + i*12
			binary.LittleEndian.PutUint16(out[off:], entry.id)
			binary.LittleEndian.PutUint16(out[off+2:], entry.kind)
			binary.LittleEndian.PutUint32(out[off+4:], entry.count)
			if len(entry.data) <= 4 {
				copy(out[off+8:off+12], entry.data)
			} else {
				binary.LittleEndian.PutUint32(out[off+8:], uint32(len(out)))
				out = append(out, entry.data...)
			}
		}
	}
	return out
}

func TestImagePrepareReadsEXIFAndAppliesOrientation(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 20, 10)), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	tiff := imageTestTIFF()
	payload := append([]byte("Exif\x00\x00"), tiff...)
	data := []byte{0xff, 0xd8, 0xff, 0xe1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
	data = append(data, payload...)
	data = append(data, encoded.Bytes()[2:]...)
	path := filepath.Join(t.TempDir(), "exif.jpg")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	img, values, orientation, profile, err := PrepareImage(path)
	if err != nil {
		t.Fatal(err)
	}
	if orientation != 6 || img.Bounds().Dx() != 10 || img.Bounds().Dy() != 20 {
		t.Fatalf("orientation=%d size=%v", orientation, img.Bounds())
	}
	if profile != "sRGB" {
		t.Fatalf("profile %q", profile)
	}
	for key, want := range map[string]string{"cameraMake": "Canon", "cameraModel": "Tiny", "creator": "Keeper", "lensModel": "Lens", "dpiWidth": "300", "dpiHeight": "300", "exposureTime": "0.008", "focalLength": "50", "isoSpeedRatings": "400", "flashFired": "false", "exposureProgram": "2", "meteringMode": "5", "whiteBalance": "0", "pixelWidth": "20", "pixelHeight": "10"} {
		if values[key] != want {
			t.Errorf("%s: %q != %q", key, values[key], want)
		}
	}
	if values["contentCreatedAt"] != "2024-01-02T03:04:05Z" {
		t.Errorf("date %q", values["contentCreatedAt"])
	}
}

func TestImageMissingEXIFDoesNotInventZeroFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 4, 8))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	_, values, orientation, profile, err := PrepareImage(path)
	if err != nil {
		t.Fatal(err)
	}
	if orientation != 1 || profile != "" {
		t.Fatalf("orientation=%d profile=%q", orientation, profile)
	}
	for _, key := range []string{"flashFired", "whiteBalance", "exposureProgram", "cameraMake"} {
		if _, ok := values[key]; ok {
			t.Errorf("invented %s", key)
		}
	}
}

func TestImageRejectsInvalidContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.jpg")
	if err := os.WriteFile(path, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(path); err == nil {
		t.Fatal("invalid image accepted")
	}
}

func TestImageEightFormatsProduceThumbnail(t *testing.T) {
	for _, ext := range []string{"jpg", "png", "gif", "webp", "tiff", "bmp", "heic", "avif"} {
		t.Run(ext, func(t *testing.T) {
			img, _, _, _, err := PrepareImage(filepath.Join("testdata", "sample."+ext))
			if err != nil {
				t.Fatal(err)
			}
			if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
				t.Fatalf("empty image %v", img.Bounds())
			}
			out := filepath.Join(t.TempDir(), "thumbnail.png")
			if err := WriteThumbnail(out, img); err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(out)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			cfg, err := png.DecodeConfig(f)
			if err != nil {
				t.Fatal(err)
			}
			if max(cfg.Width, cfg.Height) > 512 {
				t.Fatalf("thumbnail too large %dx%d", cfg.Width, cfg.Height)
			}
		})
	}
}

func TestImagePNGWithEXIFOrientation(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 20, 10))); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	payload := imageTestTIFF()
	chunk := make([]byte, 8+len(payload)+4)
	binary.BigEndian.PutUint32(chunk, uint32(len(payload)))
	copy(chunk[4:], "eXIf")
	copy(chunk[8:], payload)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	data = append(append(append([]byte{}, data[:33]...), chunk...), data[33:]...)
	path := filepath.Join(t.TempDir(), "oriented.png")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	img, metadata, orientation, _, err := PrepareImage(path)
	if err != nil {
		t.Fatal(err)
	}
	if orientation != 6 || img.Bounds().Dx() != 10 || img.Bounds().Dy() != 20 {
		t.Fatalf("orientation=%d size=%v", orientation, img.Bounds())
	}
	if metadata["whiteBalance"] != "0" || metadata["flashFired"] != "false" {
		t.Fatalf("zero-value metadata %+v", metadata)
	}
}
