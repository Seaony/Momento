package media

// maxRSS 在 Windows 上不取，返回 0 表示跳过内存上涨检查。
func maxRSS() int64 { return 0 }
