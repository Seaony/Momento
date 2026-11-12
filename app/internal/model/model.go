package model

import "time"

const SchemaVersion = 1

type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	LibraryID     string    `json:"libraryID"`
	DisplayName   string    `json:"displayName"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Color struct {
	Hex        string  `json:"hex"`
	Proportion float64 `json:"proportion"`
	Family     string  `json:"family"`
}

type Asset struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	OriginalFileName string            `json:"originalFileName"`
	Extension        string            `json:"extension"`
	MediaType        string            `json:"mediaType"`
	Size             int64             `json:"size"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	Orientation      int               `json:"orientation"`
	ColorProfile     string            `json:"colorProfile"`
	EXIF             map[string]string `json:"exif"`
	Note             string            `json:"note"`
	SourceURL        string            `json:"sourceURL"`
	RelativePath     string            `json:"relativePath"`
	Favorite         bool              `json:"favorite"`
	Trashed          bool              `json:"trashed"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	Colors           []Color           `json:"colors"`
	FolderIDs        []string          `json:"folderIDs"`
	TagIDs           []string          `json:"tagIDs"`
}

type Folder struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ParentID  string `json:"parentID"`
	SortIndex int    `json:"sortIndex"`
}

type Tag struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	NormalizedName string `json:"normalizedName"`
	ColorHex       string `json:"colorHex"`
}

type Snapshot struct {
	Assets  []Asset
	Folders []Folder
	Tags    []Tag
}

type ImportProgress struct {
	Stage       string
	Total       int
	Processed   int
	Imported    int
	Skipped     int
	CurrentFile string
}

type ImportResult struct {
	Imported int
	Skipped  int
	Restored int
	Errors   []error
}

type ImportRequest struct {
	Paths     []string
	SourceURL string
	FolderID  string
}

type ExportFormat string

const (
	ExportOriginal ExportFormat = "original"
	ExportJPEG     ExportFormat = "jpeg"
	ExportPNG      ExportFormat = "png"
)
