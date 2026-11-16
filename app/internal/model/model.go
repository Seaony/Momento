package model

import "time"

const SchemaVersion = 3

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	LibraryID     string `json:"libraryID"`
	DisplayName   string `json:"displayName"`
	// Icon 是图库图标的编号（内置图标之一），空着用应用图标。
	Icon      string    `json:"icon,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Color struct {
	Hex        string  `json:"hex"`
	Proportion float64 `json:"proportion"`
	Family     string  `json:"family"`
}

type Asset struct {
	LastUsedAt       time.Time         `json:"lastUsedAt"`
	Rotation         int               `json:"rotation"`
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
	Icon      string `json:"icon,omitempty"`
}

type Tag struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	NormalizedName string `json:"normalizedName"`
	ColorHex       string `json:"colorHex"`
}

type SmartRule struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}
type SmartFolder struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Match     string      `json:"match"`
	Rules     []SmartRule `json:"rules"`
	SortIndex int         `json:"sortIndex"`
}
type QuickAccess struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	TargetID  string `json:"targetID"`
	SortIndex int    `json:"sortIndex"`
}
type Snapshot struct {
	SmartFolders []SmartFolder
	QuickAccess  []QuickAccess
	Assets       []Asset
	Folders      []Folder
	Tags         []Tag
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
	AssetIDs []string
	Imported int
	Skipped  int
	Restored int
	Errors   []error
	// 没能导入的源文件（绝对路径），和 Errors 一一对应；自动导入靠它决定哪些源文件不能移到废纸篓。
	Failed []string
}

type ImportRequest struct {
	TagIDs         []string
	ImportKeywords bool
	Paths          []string
	SourceURL      string
	FolderID       string
}

type ExportFormat string

const (
	ExportOriginal ExportFormat = "original"
	ExportJPEG     ExportFormat = "jpeg"
	ExportPNG      ExportFormat = "png"
)
