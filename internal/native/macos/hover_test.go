//go:build darwin

package macos

import "testing"

func TestHoverActionsFollowConfiguredOrder(t *testing.T) {
	row := HistoryRow{Kind: "text", CanDelete: true}
	specs := hoverActionsFor(row, []string{"copy", "copyPlain", "delete"})
	if len(specs) != 3 {
		t.Fatalf("specs = %+v", specs)
	}
	if specs[0].Event != "copy" || specs[1].Event != "copyPlain" || specs[2].Event != "delete" {
		t.Errorf("events = %+v", specs)
	}
}

func TestHoverActionsSkipInapplicableRows(t *testing.T) {
	row := HistoryRow{Kind: "image"}
	specs := hoverActionsFor(row, []string{"copyPlain", "reveal", "openLink", "copy"})
	if len(specs) != 1 || specs[0].Event != "copy" {
		t.Fatalf("specs = %+v", specs)
	}
}

func TestHoverActionsCapAtSix(t *testing.T) {
	row := HistoryRow{Kind: "files", SubKind: "path"}
	row.CanDelete = true
	actions := []string{"paste", "copy"}
	specs := hoverActionsFor(row, actions)
	if len(specs) != len(actions) {
		t.Fatalf("specs = %+v", specs)
	}
	long := hoverActionsFor(row, []string{
		"paste", "copy", "reveal", "note", "star", "pinItem", "delete",
	})
	if len(long) != maxHoverActions {
		t.Errorf("len = %d", len(long))
	}
}

func TestHoverActionsHideProtectedDelete(t *testing.T) {
	row := HistoryRow{Kind: "text", Favorite: true, CanDelete: false}
	specs := hoverActionsFor(row, []string{"copy", "star", "delete"})
	for _, spec := range specs {
		if spec.Event == "delete" {
			t.Fatalf("delete offered for protected row: %+v", specs)
		}
	}
	if len(specs) != 2 {
		t.Errorf("specs = %+v", specs)
	}
}

func TestHoverActionReflectsItemState(t *testing.T) {
	star, ok := hoverActionFor("star", HistoryRow{Kind: "text", Favorite: true})
	if !ok || star.Symbol != "star.fill" || star.Tooltip != "取消收藏" {
		t.Errorf("star = %+v", star)
	}
	pin, ok := hoverActionFor("pinItem", HistoryRow{Kind: "text", Pinned: true})
	if !ok || pin.Symbol != "pin.fill" || pin.Tooltip != "取消置顶" {
		t.Errorf("pin = %+v", pin)
	}
}

func TestHoverActionMapsSubKindLinks(t *testing.T) {
	link, ok := hoverActionFor("openLink", HistoryRow{Kind: "text", SubKind: "url"})
	if !ok || link.Event != "open" {
		t.Errorf("link = %+v", link)
	}
	if _, ok := hoverActionFor("openLink", HistoryRow{Kind: "text", SubKind: "email"}); ok {
		t.Error("openLink applied to email row")
	}
	mail, ok := hoverActionFor("sendEmail", HistoryRow{Kind: "text", SubKind: "email"})
	if !ok || mail.Event != "open" {
		t.Errorf("mail = %+v", mail)
	}
}

func TestHoverBarFrameKeepsEqualCardInsets(t *testing.T) {
	row := CGRect{Origin: CGPoint{X: 8, Y: 24}, Size: CGSize{Width: 404, Height: 62}}
	barWidth := 146.0
	frame := hoverBarFrame(row, barWidth)
	cardRight := row.Origin.X + row.Size.Width - historyRowInsetX
	barRight := frame.Origin.X + frame.Size.Width
	cardTop := row.Origin.Y + row.Size.Height - historyRowInsetY
	barTop := frame.Origin.Y + frame.Size.Height
	if got, want := cardRight-barRight, hoverBarInset; got != want {
		t.Errorf("right inset = %v, want %v", got, want)
	}
	if got, want := cardTop-barTop, hoverBarInset; got != want {
		t.Errorf("top inset = %v, want %v", got, want)
	}
	if frame.Size.Width != barWidth || frame.Size.Height != hoverBarHeight {
		t.Errorf("toolbar size = %v, want %v×%v", frame.Size, barWidth, hoverBarHeight)
	}
}
