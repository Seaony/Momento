package assets

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestImagePaletteMergesNearbyBucketsAndSkipsTransparentPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	for i, c := range []color.NRGBA{{255, 0, 0, 255}, {240, 0, 0, 255}, {0, 0, 255, 255}, {0, 255, 0, 12}} {
		img.SetNRGBA(i, 0, c)
	}
	got := Palette(img)
	if len(got) != 2 || got[0].Hex != "#F70000" || got[0].Family != "red" || got[1].Hex != "#0000FF" {
		t.Fatalf("palette %+v", got)
	}
	if math.Abs(got[0].Proportion-2.0/3) > 1e-9 || math.Abs(got[1].Proportion-1.0/3) > 1e-9 {
		t.Fatalf("proportions %+v", got)
	}
	if got := Palette(image.NewNRGBA(image.Rect(0, 0, 2, 2))); len(got) != 0 {
		t.Fatalf("transparent palette %+v", got)
	}
}

func TestImagePaletteCapsEightAndUsesStableTieOrder(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 1))
	for i := range 16 {
		img.SetNRGBA(i, 0, color.NRGBA{uint8((i % 4) * 85), uint8((i / 4) * 85), 127, 255})
	}
	got := Palette(img)
	if len(got) != 8 {
		t.Fatalf("colors %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Hex > got[i].Hex {
			t.Fatal("unstable tie order")
		}
	}
}

func TestImageColorFamilies(t *testing.T) {
	for hex, want := range map[string]string{
		"#000000": "black", "#FFFFFF": "white", "#888888": "gray", "#D9CCB8": "beige", "#804000": "brown", "#707020": "olive",
		"#FF0000": "red", "#FF5000": "coral", "#FF8000": "orange", "#FFBB00": "amber", "#FFFF00": "yellow", "#BFFF00": "lime",
		"#00FF00": "green", "#00FF99": "mint", "#00FFEE": "teal", "#00AAFF": "cyan", "#0080FF": "sky", "#0000FF": "blue",
		"#5500FF": "indigo", "#9900FF": "violet", "#D500FF": "purple", "#FF00CC": "magenta", "#FF0080": "pink", "#FF0040": "rose",
		"bad": "", "#ZZZZZZ": "",
	} {
		if got := ColorFamily(hex); got != want {
			t.Errorf("%s: %s != %s", hex, got, want)
		}
	}
}
