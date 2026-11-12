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
	for _, rule := range folder.Rules {
		valid := false
		switch rule.Field {
		case "name", "note":
			valid = slices.Contains([]string{"contains", "equals"}, rule.Operator) && strings.TrimSpace(rule.Value) != ""
		case "extension", "tag", "color":
			valid = rule.Operator == "equals" && strings.TrimSpace(rule.Value) != ""
		case "width", "height", "size":
			n, err := strconv.ParseInt(rule.Value, 10, 64)
			valid = err == nil && n >= 0 && slices.Contains([]string{"eq", "gte", "lte"}, rule.Operator)
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
	return matchesValidatedSmart(asset, folder)
}

func matchesValidatedSmart(asset model.Asset, folder model.SmartFolder) bool {
	for _, rule := range folder.Rules {
		matched := matchesSmartRule(asset, rule)
		if folder.Match == "all" && !matched {
			return false
		}
		if folder.Match == "any" && matched {
			return true
		}
	}
	return folder.Match == "all"
}

func matchesSmartRule(asset model.Asset, rule model.SmartRule) bool {
	value := strings.ToLower(strings.TrimSpace(rule.Value))
	switch rule.Field {
	case "name", "note":
		text := asset.Name
		if rule.Field == "note" {
			text = asset.Note
		}
		text = strings.ToLower(text)
		if rule.Operator == "contains" {
			return strings.Contains(text, value)
		}
		return text == value
	case "extension":
		return normalizedExtension(asset.Extension) == normalizedExtension(value)
	case "tag":
		return slices.Contains(asset.TagIDs, rule.Value)
	case "color":
		return slices.ContainsFunc(asset.Colors, func(c model.Color) bool { return strings.EqualFold(c.Family, value) || strings.EqualFold(c.Hex, value) })
	case "width", "height", "size":
		target, err := strconv.ParseInt(rule.Value, 10, 64)
		if err != nil {
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
		switch rule.Operator {
		case "eq":
			return actual == target
		case "gte":
			return actual >= target
		case "lte":
			return actual <= target
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
