package assets

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	metajpeg "github.com/evanoberholster/imagemeta/meta/jpeg"
	"github.com/evanoberholster/imagemeta/meta/xmp"
	"github.com/google/uuid"
)

// ReadKeywords 合并 IPTC 与 XMP 关键字，沿用图库标签的去重规则。
func ReadKeywords(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var keywords []string
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".jpg" || extension == ".jpeg" {
		metadata, err := metajpeg.ScanMetadata(f)
		if err != nil {
			return nil, err
		}
		if metadata.IPTC != nil {
			keywords = append(keywords, metadata.IPTC.Keywords...)
		}
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var values xmp.XMP
	if extension == ".png" {
		values, err = pngKeywords(f)
	} else {
		values, err = xmp.Parse(f)
	}
	if err != nil && !errors.Is(err, xmp.ErrNoXMP) {
		return nil, err
	}
	keywords = append(keywords, values.DC.Subject...)
	var result []string
	seen := map[string]bool{}
	for _, name := range keywords {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !utf8.ValidString(name) || strings.ContainsRune(name, 0) {
			return nil, i18n.NewError("error_keyword_encoding")
		}
		normalized := library.NormalizeTag(name)
		if !seen[normalized] {
			result = append(result, name)
			seen[normalized] = true
		}
	}
	return result, nil
}

func pngKeywords(r io.ReadSeeker) (xmp.XMP, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return xmp.XMP{}, err
	}
	if !bytes.Equal(header[:], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return xmp.XMP{}, i18n.NewError("error_keyword_metadata")
	}
	for {
		if _, err := io.ReadFull(r, header[:]); err != nil {
			return xmp.XMP{}, err
		}
		size := int64(binary.BigEndian.Uint32(header[:4]))
		kind := string(header[4:])
		if kind == "iTXt" {
			if size > 4*1024*1024 {
				return xmp.XMP{}, i18n.NewError("error_keyword_too_large")
			}
			data := make([]byte, size)
			if _, err := io.ReadFull(r, data); err != nil {
				return xmp.XMP{}, err
			}
			keyword, remaining, found := bytes.Cut(data, []byte{0})
			if found && string(keyword) == "XML:com.adobe.xmp" {
				if len(remaining) < 2 || remaining[0] > 1 || remaining[1] != 0 {
					return xmp.XMP{}, i18n.NewError("error_keyword_metadata")
				}
				compressed := remaining[0] == 1
				_, remaining, found = bytes.Cut(remaining[2:], []byte{0})
				if !found {
					return xmp.XMP{}, io.ErrUnexpectedEOF
				}
				_, packet, found := bytes.Cut(remaining, []byte{0})
				if !found {
					return xmp.XMP{}, io.ErrUnexpectedEOF
				}
				if !compressed {
					return xmp.ParseXmp(bytes.NewReader(packet))
				}
				reader, err := zlib.NewReader(bytes.NewReader(packet))
				if err != nil {
					return xmp.XMP{}, err
				}
				defer reader.Close()
				expanded, err := io.ReadAll(io.LimitReader(reader, 4*1024*1024+1))
				if err != nil {
					return xmp.XMP{}, err
				}
				if len(expanded) > 4*1024*1024 {
					return xmp.XMP{}, i18n.NewError("error_keyword_too_large")
				}
				return xmp.ParseXmp(bytes.NewReader(expanded))
			}
			size = 0
		}
		if _, err := r.Seek(size+4, io.SeekCurrent); err != nil {
			return xmp.XMP{}, err
		}
		if kind == "IEND" {
			return xmp.XMP{}, xmp.ErrNoXMP
		}
	}
}

func attachKeywords(ctx context.Context, tx *sql.Tx, assetID string, keywords []string) error {
	for _, name := range keywords {
		normalized := library.NormalizeTag(name)
		if _, err := tx.ExecContext(ctx, "INSERT INTO tags(id,name,normalized_name) VALUES(?,?,?) ON CONFLICT(normalized_name) DO NOTHING", uuid.NewString(), name, normalized); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO asset_tags(asset_id,tag_id) SELECT ?,id FROM tags WHERE normalized_name=?", assetID, normalized); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) verifyDuplicate(ctx context.Context, relative, expected string) error {
	path, err := s.assetPath(relative)
	if err != nil {
		return i18n.WrapError(err, "error_duplicate_copy")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	actual, err := hashReader(ctx, f)
	if err != nil {
		return err
	}
	if actual != expected {
		return i18n.NewError("error_duplicate_damaged")
	}
	return nil
}
