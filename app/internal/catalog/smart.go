package catalog

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func ValidateSmartFolder(folder model.SmartFolder) error {
	if !slices.Contains([]string{"all", "any"}, folder.Match) || len(folder.Rules) == 0 {
		return i18n.NewError("error_smart_rule")
	}
	for i := range folder.Rules {
		rule := &folder.Rules[i]
		valid := false
		switch rule.Field {
		case "name", "note", "source":
			valid = slices.Contains(TextOperators, rule.Operator) && strings.TrimSpace(rule.Value) != ""
		case "extension", "tag", "color":
			valid = rule.Operator == "equals" && strings.TrimSpace(rule.Value) != ""
		case "width", "height", "size", "duration", "rating":
			low, high, err := numericRange(rule)
			valid = err == nil && low >= 0 && low <= high && (rule.Field != "rating" || high <= 5)
		case "added":
			_, err := parseRuleDate(rule.Value)
			valid = err == nil && slices.Contains([]string{"after", "before"}, rule.Operator)
		}
		if !valid {
			return i18n.NewError("error_smart_rule")
		}
	}
	return nil
}

// TextOperators 是名称、备注、来源链接能用的比较方式。
var TextOperators = []string{"contains", "not_contains", "equals", "starts_with", "ends_with"}

// numericRange 把数值条件的值读成范围：「介于」存成「小,大」两端都包含，其余是一个数（low == high）。
func numericRange(rule *model.SmartRule) (int64, int64, error) {
	if rule.Operator == "between" {
		left, right, ok := strings.Cut(rule.Value, ",")
		if !ok {
			return 0, 0, strconv.ErrSyntax
		}
		low, err := strconv.ParseInt(strings.TrimSpace(left), 10, 64)
		if err != nil {
			return 0, 0, err
		}
		high, err := strconv.ParseInt(strings.TrimSpace(right), 10, 64)
		return low, high, err
	}
	if !slices.Contains([]string{"eq", "gte", "lte"}, rule.Operator) {
		return 0, 0, strconv.ErrSyntax
	}
	n, err := strconv.ParseInt(rule.Value, 10, 64)
	return n, n, err
}

func parseRuleDate(value string) (time.Time, error) {
	if day, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return day, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

func MatchesSmart(asset model.Asset, folder model.SmartFolder) bool {
	if asset.Trashed || ValidateSmartFolder(folder) != nil {
		return false
	}
	return matchesValidatedSmart(&asset, &folder)
}

func matchesValidatedSmart(asset *model.Asset, folder *model.SmartFolder) bool {
	for i := range folder.Rules {
		matched := matchesSmartRule(asset, &folder.Rules[i])
		if folder.Match == "all" && !matched {
			return false
		}
		if folder.Match == "any" && matched {
			return true
		}
	}
	return folder.Match == "all"
}

func matchesSmartRule(asset *model.Asset, rule *model.SmartRule) bool {
	value := strings.ToLower(strings.TrimSpace(rule.Value))
	switch rule.Field {
	case "name", "note", "source":
		text := asset.Name
		switch rule.Field {
		case "note":
			text = asset.Note
		case "source":
			text = asset.SourceURL
		}
		text = strings.ToLower(text)
		switch rule.Operator {
		case "contains":
			return strings.Contains(text, value)
		case "not_contains":
			return !strings.Contains(text, value)
		case "starts_with":
			return strings.HasPrefix(text, value)
		case "ends_with":
			return strings.HasSuffix(text, value)
		}
		return text == value
	case "extension":
		return normalizedExtension(asset.Extension) == normalizedExtension(value)
	case "tag":
		return slices.Contains(asset.TagIDs, rule.Value)
	case "color":
		return slices.ContainsFunc(asset.Colors, func(c model.Color) bool { return strings.EqualFold(c.Family, value) || strings.EqualFold(c.Hex, value) })
	case "width", "height", "size", "duration", "rating":
		low, high, err := numericRange(rule)
		if err != nil {
			return false
		}
		// 时长只对视频和实况照片有意义，图片的 0 不算「时长 ≤ N」。
		if rule.Field == "duration" && asset.Kind != model.KindVideo && asset.Kind != model.KindLive {
			return false
		}
		width, height := asset.Width, asset.Height
		if asset.Rotation == 90 || asset.Rotation == 270 {
			width, height = height, width
		}
		actual := asset.Size
		if rule.Field == "width" {
			actual = int64(width)
		}
		if rule.Field == "height" {
			actual = int64(height)
		}
		if rule.Field == "duration" {
			actual = asset.DurationMS
		}
		if rule.Field == "rating" {
			actual = int64(asset.Rating)
		}
		switch rule.Operator {
		case "eq":
			return actual == low
		case "gte":
			return actual >= low
		case "lte":
			return actual <= low
		case "between":
			return actual >= low && actual <= high
		}
	case "added":
		target, err := parseRuleDate(rule.Value)
		if err != nil {
			return false
		}
		if rule.Operator == "after" {
			return !asset.CreatedAt.Before(target)
		}
		// 日期按整天包含；带时分秒时按指定瞬间包含。
		if len(rule.Value) == len("2006-01-02") {
			return asset.CreatedAt.Before(target.AddDate(0, 0, 1))
		}
		return !asset.CreatedAt.After(target)
	}
	return false
}
