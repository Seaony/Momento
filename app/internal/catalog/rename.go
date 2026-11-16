package catalog

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

// 批量重命名格式里能用的占位符。
const (
	TokenName   = "{name}"
	TokenNumber = "{n}"
	TokenDate   = "{date}"
	TokenTags   = "{tags}"
)

// RenameRule 是批量重命名的规则：Replace 为 false 时按 Template 生成名称，为 true 时在原名里查找替换。
type RenameRule struct {
	Replace  bool
	Template string
	// Start 是第一个素材的序号；序号按最大的那个补零，比如 12 个素材是 01～12。
	Start int
	Find  string
	With  string
	Regex bool
}

// BatchNames 按规则算出每个素材的新名称，顺序和 assets 一样，前后空白去掉。tagNames 是标签 ID 到名称。
func BatchNames(assets []model.Asset, tagNames map[string]string, rule RenameRule) ([]string, error) {
	names := make([]string, len(assets))
	if rule.Replace {
		var pattern *regexp.Regexp
		if rule.Regex && rule.Find != "" {
			var err error
			if pattern, err = regexp.Compile(rule.Find); err != nil {
				// 输入时每帧都会算一遍，写错是正常的编辑过程，只在对话框里提示，不记日志。
				return nil, i18n.NewError("error_rename_regex")
			}
		}
		for i, asset := range assets {
			switch {
			case rule.Find == "":
				names[i] = asset.Name
			case pattern != nil:
				names[i] = pattern.ReplaceAllString(asset.Name, rule.With)
			default:
				names[i] = strings.ReplaceAll(asset.Name, rule.Find, rule.With)
			}
			names[i] = strings.TrimSpace(names[i])
		}
		return names, nil
	}
	width := len(strconv.Itoa(rule.Start + max(len(assets)-1, 0)))
	for i, asset := range assets {
		tags := make([]string, 0, len(asset.TagIDs))
		for _, id := range asset.TagIDs {
			if name := tagNames[id]; name != "" {
				tags = append(tags, name)
			}
		}
		number := strconv.Itoa(rule.Start + i)
		number = strings.Repeat("0", max(width-len(number), 0)) + number
		date := ""
		if !asset.CreatedAt.IsZero() {
			date = asset.CreatedAt.Local().Format("2006-01-02")
		}
		names[i] = strings.TrimSpace(strings.NewReplacer(TokenName, asset.Name, TokenNumber, number, TokenDate, date, TokenTags, strings.Join(tags, " ")).Replace(rule.Template))
	}
	return names, nil
}
