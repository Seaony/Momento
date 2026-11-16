package catalog

import (
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func TestBatchNames(t *testing.T) {
	added := time.Date(2026, 3, 9, 12, 0, 0, 0, time.Local)
	assets := make([]model.Asset, 12)
	for i := range assets {
		assets[i] = model.Asset{ID: string(rune('a' + i)), Name: "IMG_100" + string(rune('0'+i%10)), CreatedAt: added}
	}
	assets[0].TagIDs = []string{"t1", "gone", "t2"}
	tags := map[string]string{"t1": "猫", "t2": "夜景"}

	names, err := BatchNames(assets, tags, RenameRule{Template: "旅行-{n} {name} {date} {tags}", Start: 1})
	if err != nil {
		t.Fatal(err)
	}
	if names[0] != "旅行-01 IMG_1000 2026-03-09 猫 夜景" || names[11] != "旅行-12 IMG_1001 2026-03-09" {
		t.Fatalf("格式结果不对：%q / %q", names[0], names[11])
	}
	if names, _ := BatchNames(assets[:3], nil, RenameRule{Template: "{n}", Start: 9}); !slices.Equal(names, []string{"09", "10", "11"}) {
		t.Fatalf("序号补零不对：%v", names)
	}
	if names, _ := BatchNames(assets[:2], nil, RenameRule{Template: "  "}); names[0] != "" {
		t.Fatalf("全是空白的格式应该得到空名称：%q", names[0])
	}

	names, err = BatchNames(assets[:2], nil, RenameRule{Replace: true, Find: "IMG_", With: "照片 "})
	if err != nil || !slices.Equal(names, []string{"照片 1000", "照片 1001"}) {
		t.Fatalf("普通替换：%v %v", names, err)
	}
	if names, _ := BatchNames(assets[:1], nil, RenameRule{Replace: true, Find: "1.0", With: "x"}); names[0] != "IMG_1000" {
		t.Fatalf("普通替换不该把点当通配符：%v", names)
	}
	names, err = BatchNames(assets[:2], nil, RenameRule{Replace: true, Regex: true, Find: `IMG_(\d+)`, With: "${1}号"})
	if err != nil || !slices.Equal(names, []string{"1000号", "1001号"}) {
		t.Fatalf("正则替换：%v %v", names, err)
	}
	if _, err := BatchNames(assets, nil, RenameRule{Replace: true, Regex: true, Find: "("}); err == nil {
		t.Fatal("写错的正则没有报错")
	}
	if names, _ := BatchNames(assets[:1], nil, RenameRule{Replace: true}); names[0] != "IMG_1000" {
		t.Fatalf("查找为空时应保留原名：%v", names)
	}
}
