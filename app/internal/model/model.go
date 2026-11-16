package model

import "time"

const SchemaVersion = 6

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
	// Kind 是素材种类：KindImage、KindVideo、KindLive（实况照片：一张图配一段短视频）。
	Kind string `json:"kind"`
	// DurationMS 是视频时长（实况照片是那段视频的时长），图片为 0。
	DurationMS int64 `json:"durationMS"`
	// MotionPath 是实况照片那段视频在图库包里的相对路径，其他素材为空。
	MotionPath string `json:"motionPath"`
	// Rating 是星级评分，0 是没评分，1～5 星。
	Rating int `json:"rating"`
	// Annotations 是图上的区域批注，按添加时间排。
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Annotation 是图上的一块区域批注。X、Y、W、H 是相对图（已按 EXIF 转正、没手动旋转时）宽高的比例，0～1。
type Annotation struct {
	ID        string    `json:"id"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	W         float64   `json:"w"`
	H         float64   `json:"h"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

const (
	KindImage = "image"
	KindVideo = "video"
	KindLive  = "live"
)

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
	// Motion 是配进实况照片、跟着图片一起导入（或一起算重复）的视频源文件；它们不单独计入 Imported 和 Skipped。
	Motion []string
}

type ImportRequest struct {
	TagIDs         []string
	ImportKeywords bool
	Paths          []string
	SourceURL      string
	FolderID       string
	// TagNames 按名字打标签，没有就新建（浏览器扩展传来的标签）。
	TagNames []string
	// Name、Note 只写给这次新入库的素材；图库里已有的不改。
	Name, Note string
}

type ExportFormat string

const (
	ExportOriginal ExportFormat = "original"
	ExportJPEG     ExportFormat = "jpeg"
	ExportPNG      ExportFormat = "png"
)

// ExportOptions 是一次导出的设置，也是导出预设保存的内容。
// MaxEdge 是转成 JPEG/PNG 时最长边的上限（像素），0 是原尺寸；比上限小的图不放大。原格式导出不改尺寸。
type ExportOptions struct {
	Format  ExportFormat `json:"format"`
	Quality float64      `json:"quality"`
	MaxEdge int          `json:"maxEdge,omitempty"`
	Motion  bool         `json:"motion,omitempty"`
}
