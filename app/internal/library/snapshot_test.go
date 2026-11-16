package library

import (
	"slices"
	"testing"
)

// 快照不按 ID 排素材，但每个素材的文件夹、标签 ID 必须升序，颜色按 sort_index 排，且颜色互不串到别的素材。
func TestSnapshotKeepsMemberAndColorOrder(t *testing.T) {
	l := newTestLibrary(t)
	for _, id := range []string{"z", "a", "m"} {
		seedAsset(t, l, id)
	}
	for _, query := range []string{
		"INSERT INTO folders(id,name) VALUES('f2','二'),('f1','一'),('f3','三')",
		"INSERT INTO tags(id,name,normalized_name) VALUES('t2','二','二'),('t1','一','一')",
		"INSERT INTO asset_folders(asset_id,folder_id) VALUES('z','f3'),('z','f1'),('a','f2'),('z','f2')",
		"INSERT INTO asset_tags(asset_id,tag_id) VALUES('m','t2'),('m','t1')",
		// 颜色故意乱序写入：先写后面的素材、同一素材先写排在后面的颜色。
		"INSERT INTO asset_colors(asset_id,hex,proportion,family,sort_index) VALUES('m','#000002',0.2,'blue',1),('z','#000001',0.5,'red',0),('m','#000001',0.8,'red',0),('a','#000003',1,'green',0),('m','#000003',0.1,'gray',2)",
	} {
		if _, err := l.DB.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	s := snapshot(t, l)
	byID := map[string]int{}
	for i, asset := range s.Assets {
		byID[asset.ID] = i
	}
	z, a, m := s.Assets[byID["z"]], s.Assets[byID["a"]], s.Assets[byID["m"]]
	if !slices.Equal(z.FolderIDs, []string{"f1", "f2", "f3"}) || !slices.Equal(a.FolderIDs, []string{"f2"}) || m.FolderIDs != nil {
		t.Fatalf("文件夹 %v %v %v", z.FolderIDs, a.FolderIDs, m.FolderIDs)
	}
	if !slices.Equal(m.TagIDs, []string{"t1", "t2"}) || z.TagIDs != nil {
		t.Fatalf("标签 %v %v", m.TagIDs, z.TagIDs)
	}
	hexes := func(i int) []string {
		var out []string
		for _, color := range s.Assets[i].Colors {
			out = append(out, color.Hex)
		}
		return out
	}
	if got := hexes(byID["m"]); !slices.Equal(got, []string{"#000001", "#000002", "#000003"}) {
		t.Fatalf("颜色没有按 sort_index 排：%v", got)
	}
	if got := hexes(byID["z"]); !slices.Equal(got, []string{"#000001"}) {
		t.Fatalf("颜色串到了别的素材：%v", got)
	}
	// 颜色共用底层数组，往一个素材的颜色后追加不能改到别的素材。
	before := slices.Clone(s.Assets[byID["a"]].Colors)
	for i := range s.Assets {
		s.Assets[i].Colors = append(s.Assets[i].Colors, s.Assets[i].Colors...)
	}
	if got := s.Assets[byID["a"]].Colors; len(got) != 2 || got[0] != before[0] {
		t.Fatalf("追加颜色改到了别的素材：%v", got)
	}
}
