//go:build darwin

package macos

import "testing"

func testScreens() []ScreenBounds {
	main := CGRect{Origin: CGPoint{X: 0, Y: 0}, Size: CGSize{Width: 1512, Height: 982}}
	mainVisible := CGRect{Origin: CGPoint{X: 0, Y: 0}, Size: CGSize{Width: 1512, Height: 950}}
	second := CGRect{Origin: CGPoint{X: 1512, Y: 0}, Size: CGSize{Width: 1920, Height: 1080}}
	return []ScreenBounds{
		{Frame: main, Visible: mainVisible},
		{Frame: second, Visible: second},
	}
}

func TestPanelOriginFollowsCursor(t *testing.T) {
	size := CGSize{Width: 420, Height: 560}
	origin := panelOrigin("followCursor", size, CGPoint{X: 800, Y: 700}, testScreens(), nil)
	if origin.X != 800 || origin.Y != 700-560 {
		t.Fatalf("origin = %+v", origin)
	}
}

func TestPanelOriginClampsToScreenEdge(t *testing.T) {
	size := CGSize{Width: 420, Height: 560}
	origin := panelOrigin("followCursor", size, CGPoint{X: 1500, Y: 100}, testScreens(), nil)
	if origin.X != 1512-420-panelScreenMargin {
		t.Fatalf("x = %v", origin.X)
	}
	if origin.Y != panelScreenMargin {
		t.Fatalf("y = %v", origin.Y)
	}
}

func TestPanelOriginCentersOnPointerScreen(t *testing.T) {
	size := CGSize{Width: 420, Height: 560}
	origin := panelOrigin("center", size, CGPoint{X: 2000, Y: 500}, testScreens(), nil)
	if origin.X != 1512+(1920-420)/2 || origin.Y != (1080-560)/2 {
		t.Fatalf("origin = %+v", origin)
	}
}

func TestPanelOriginRememberFallsBackWhenOffScreen(t *testing.T) {
	size := CGSize{Width: 420, Height: 560}
	saved := CGPoint{X: 5000, Y: 5000}
	origin := panelOrigin("remember", size, CGPoint{X: 800, Y: 500}, testScreens(), &saved)
	if origin.X != (1512-420)/2 || origin.Y != (950-560)/2 {
		t.Fatalf("origin = %+v", origin)
	}
}

func TestPanelOriginRememberRestoresSavedPosition(t *testing.T) {
	size := CGSize{Width: 420, Height: 560}
	saved := CGPoint{X: 300, Y: 200}
	origin := panelOrigin("remember", size, CGPoint{X: 800, Y: 500}, testScreens(), &saved)
	if origin.X != 300 || origin.Y != 200 {
		t.Fatalf("origin = %+v", origin)
	}
}
