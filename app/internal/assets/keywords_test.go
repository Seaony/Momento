package assets

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
)

const keywordPacket = `<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"><rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:subject><rdf:Bag><rdf:li> Design </rdf:li><rdf:li>参考</rdf:li><rdf:li>A &amp; B</rdf:li><rdf:li>ＤＥＳＩＧＮ</rdf:li></rdf:Bag></dc:subject></rdf:Description></rdf:RDF></x:xmpmeta>`

func jpegWithKeywords(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("testdata", "sample.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	iptc := []byte{}
	for _, word := range []string{"design", " Photo ", "参考"} {
		iptc = append(iptc, 0x1c, 2, 25, byte(len(word)>>8), byte(len(word)))
		iptc = append(iptc, word...)
	}
	photoshop := append([]byte("Photoshop 3.0\x008BIM\x04\x04\x00\x00"), make([]byte, 4)...)
	binary.BigEndian.PutUint32(photoshop[len(photoshop)-4:], uint32(len(iptc)))
	photoshop = append(photoshop, iptc...)
	if len(iptc)%2 == 1 {
		photoshop = append(photoshop, 0)
	}
	segment := func(marker byte, payload []byte) []byte {
		result := []byte{0xff, marker, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
		return append(result, payload...)
	}
	data := append([]byte{}, source[:2]...)
	data = append(data, segment(0xed, photoshop)...)
	data = append(data, segment(0xe1, append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte(keywordPacket)...))...)
	data = append(data, source[2:]...)
	path := filepath.Join(t.TempDir(), "keywords.jpg")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func pngWithKeywords(t *testing.T, compressed bool) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("testdata", "sample.png"))
	if err != nil {
		t.Fatal(err)
	}
	packet := []byte(keywordPacket)
	flag := byte(0)
	if compressed {
		var buffer bytes.Buffer
		w := zlib.NewWriter(&buffer)
		if _, err = w.Write(packet); err != nil {
			t.Fatal(err)
		}
		if err = w.Close(); err != nil {
			t.Fatal(err)
		}
		packet = buffer.Bytes()
		flag = 1
	}
	payload := append([]byte("XML:com.adobe.xmp\x00"), flag, 0, 0, 0)
	payload = append(payload, packet...)
	chunk := make([]byte, len(payload)+12)
	binary.BigEndian.PutUint32(chunk, uint32(len(payload)))
	copy(chunk[4:], "iTXt")
	copy(chunk[8:], payload)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	data := append(append(append([]byte{}, source[:33]...), chunk...), source[33:]...)
	path := filepath.Join(t.TempDir(), "keywords.png")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadKeywordsMergesIPTCAndXMP(t *testing.T) {
	words, err := ReadKeywords(jpegWithKeywords(t))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"design", "Photo", "参考", "A & B"}; !reflect.DeepEqual(words, want) {
		t.Fatalf("关键字错误: %v", words)
	}
	for _, compressed := range []bool{false, true} {
		words, err = ReadKeywords(pngWithKeywords(t, compressed))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(words, []string{"Design", "参考", "A & B"}) {
			t.Fatalf("PNG关键字错误: %v", words)
		}
	}
}

func TestKeywordImportIsOptInAndDeduplicatesTags(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	path := jpegWithKeywords(t)
	first, err := s.Import(ctx, model.ImportRequest{Paths: []string{path}}, nil)
	if err != nil || len(first.AssetIDs) != 1 {
		t.Fatalf("首次导入: %+v %v", first, err)
	}
	if len(snapshot(t, ctx, lib).Tags) != 0 {
		t.Fatal("关闭关键字选项却创建标签")
	}
	tag, err := lib.CreateTag(ctx, "DESIGN")
	if err != nil {
		t.Fatal(err)
	}
	manual, err := lib.CreateTag(ctx, "Manual")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Import(ctx, model.ImportRequest{Paths: []string{path, path}, ImportKeywords: true, TagIDs: []string{manual.ID}}, nil)
	if err != nil || second.Imported != 0 || second.Skipped != 1 || !reflect.DeepEqual(second.AssetIDs, first.AssetIDs) {
		t.Fatalf("重复导入: %+v %v", second, err)
	}
	got := snapshot(t, ctx, lib)
	if len(got.Tags) != 5 || len(got.Assets) != 1 || len(got.Assets[0].TagIDs) != 5 {
		t.Fatalf("关键字没有合并到素材: %+v", got)
	}
	found := false
	for _, id := range got.Assets[0].TagIDs {
		if id == tag.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("没有复用已有的规范化标签")
	}
	if err := s.Trash(ctx, first.AssetIDs); err != nil {
		t.Fatal(err)
	}
	third, err := s.Import(ctx, model.ImportRequest{Paths: []string{path}, ImportKeywords: true}, nil)
	if err != nil || third.Imported != 1 || third.Restored != 1 || !reflect.DeepEqual(third.AssetIDs, first.AssetIDs) {
		t.Fatalf("恢复结果错误: %+v %v", third, err)
	}
}

func TestKeywordTagsRollBackWithImportFailure(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	path := pngWithKeywords(t, true)
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{path}, ImportKeywords: true, TagIDs: []string{"missing"}}, nil)
	if err == nil || result.Imported != 0 || len(result.AssetIDs) != 0 {
		t.Fatalf("错误事务未清空结果: %+v %v", result, err)
	}
	got := snapshot(t, ctx, lib)
	if len(got.Assets) != 0 || len(got.Tags) != 0 || len(storedFiles(t, lib.Root)) != 0 {
		t.Fatal("失败留下素材、标签或文件")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("失败改动了源文件")
	}
}

func TestKeywordReadingKeepsAllEightFormatsImportable(t *testing.T) {
	for _, extension := range []string{"jpg", "png", "gif", "webp", "tiff", "bmp", "heic", "avif"} {
		t.Run(extension, func(t *testing.T) {
			ctx, _, s := testLibrary(t)
			result, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample."+extension)}, ImportKeywords: true}, nil)
			if err != nil || result.Imported != 1 {
				t.Fatalf("无关键字图片导入失败: %+v %v", result, err)
			}
		})
	}
}

func TestDuplicateRequiresAnIntactStoredCopy(t *testing.T) {
	for _, failure := range []string{"missing", "damaged"} {
		t.Run(failure, func(t *testing.T) {
			ctx, lib, s := testLibrary(t)
			path := pngWithKeywords(t, false)
			if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{path}}, nil); err != nil {
				t.Fatal(err)
			}
			asset := snapshot(t, ctx, lib).Assets[0]
			stored := filepath.Join(lib.Root, filepath.FromSlash(asset.RelativePath))
			if failure == "missing" {
				if err := os.Remove(stored); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(stored, []byte("damaged"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			result, err := s.Import(ctx, model.ImportRequest{Paths: []string{path}, ImportKeywords: true}, nil)
			if err == nil || result.Skipped != 0 || len(result.AssetIDs) != 0 {
				t.Fatalf("缺少完整副本仍被当作成功重复: %+v %v", result, err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatal("失败修改了源文件")
			}
			if len(snapshot(t, ctx, lib).Tags) != 0 {
				t.Fatal("失败仍写入标签")
			}
		})
	}
}

func TestMalformedKeywordsFailWithoutChangingLibrary(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	path := jpegWithKeywords(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("参考"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 1)
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{path}, ImportKeywords: true}, nil)
	if err == nil || result.Imported != 0 || len(snapshot(t, ctx, lib).Assets) != 0 {
		t.Fatalf("错误关键字没有阻止导入: %+v %v", result, err)
	}
}

func TestImportURLWithRequestPreservesDestinationKeywordsAndTags(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	path := jpegWithKeywords(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	folder, err := lib.CreateFolder(ctx, "Destination", "")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := lib.CreateTag(ctx, "Chosen")
	if err != nil {
		t.Fatal(err)
	}
	request := model.ImportRequest{Paths: []string{"ignored"}, SourceURL: "ignored", FolderID: folder.ID, TagIDs: []string{tag.ID}, ImportKeywords: true}
	result, err := s.ImportURLWithRequest(ctx, server.URL+"/image", "https://example.com/source", request)
	if err != nil || result.Imported != 1 || len(result.AssetIDs) != 1 {
		t.Fatalf("远程请求导入失败: %+v %v", result, err)
	}
	got := snapshot(t, ctx, lib)
	if len(got.Assets) != 1 || len(got.Assets[0].TagIDs) != 5 || !reflect.DeepEqual(got.Assets[0].FolderIDs, []string{folder.ID}) || got.Assets[0].SourceURL != "https://example.com/source" {
		t.Fatalf("远程请求选项丢失: %+v", got)
	}
}
