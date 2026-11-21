//go:build darwin

package macos

import "testing"

func TestSettingsWindowContentHeight(t *testing.T) {
	tests := []struct {
		name string
		page float64
		max  float64
		want float64
	}{
		{name: "minimum", page: 80, max: 900, want: settingsMinimumWindowHeight},
		{name: "content measured", page: 460, max: 900, want: settingsNavigationHeight + 460},
		{name: "maximum", page: 1200, max: settingsMaximumContentHeight, want: settingsMaximumContentHeight},
		{name: "small screen", page: 600, max: 360, want: 360},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := settingsWindowContentHeight(test.page, test.max); got != test.want {
				t.Fatalf("settingsWindowContentHeight(%v, %v) = %v; want %v", test.page, test.max, got, test.want)
			}
		})
	}
}

func TestSettingsFrameAtHeightKeepsTopAndFitsScreen(t *testing.T) {
	visible := CGRect{Origin: CGPoint{X: 0, Y: 0}, Size: CGSize{Width: 1440, Height: 900}}
	frame := CGRect{Origin: CGPoint{X: 400, Y: 200}, Size: CGSize{Width: 480, Height: 660}}
	resized := settingsFrameAtHeight(frame, 500, visible)
	if resized.Size.Width != frame.Size.Width || resized.Size.Height != 500 {
		t.Fatalf("resized size = %+v; width must stay %v and height 500", resized.Size, frame.Size.Width)
	}
	if top := resized.Origin.Y + resized.Size.Height; top != frame.Origin.Y+frame.Size.Height {
		t.Fatalf("top edge moved to %v; want %v", top, frame.Origin.Y+frame.Size.Height)
	}

	offscreen := settingsFrameAtHeight(CGRect{Origin: CGPoint{X: 10, Y: 850}, Size: CGSize{Width: 480, Height: 300}}, 700, visible)
	if offscreen.Origin.Y < settingsScreenMargin || offscreen.Origin.Y+offscreen.Size.Height > visible.Size.Height-settingsScreenMargin {
		t.Fatalf("resized frame extends beyond the visible screen: %+v", offscreen)
	}
}
