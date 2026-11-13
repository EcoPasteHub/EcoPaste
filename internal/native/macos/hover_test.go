//go:build darwin

package macos

import "testing"

func TestHoverActionsFollowConfiguredOrder(t *testing.T) {
	row := HistoryRow{Kind: "text"}
	specs := hoverActionsFor(row, []string{"copy", "pastePlain", "delete"})
	if len(specs) != 3 {
		t.Fatalf("specs = %+v", specs)
	}
	if specs[0].Event != "copy" || specs[1].Event != "pastePlain" || specs[2].Event != "delete" {
		t.Errorf("events = %+v", specs)
	}
}

func TestHoverActionsSkipInapplicableRows(t *testing.T) {
	row := HistoryRow{Kind: "image"}
	specs := hoverActionsFor(row, []string{"pastePlain", "pastePath", "reveal", "openLink", "copy"})
	if len(specs) != 1 || specs[0].Event != "copy" {
		t.Fatalf("specs = %+v", specs)
	}
}

func TestHoverActionsCapAtSix(t *testing.T) {
	row := HistoryRow{Kind: "files", SubKind: "path"}
	actions := []string{"paste", "pastePath", "copy"}
	specs := hoverActionsFor(row, actions)
	if len(specs) != len(actions) {
		t.Fatalf("specs = %+v", specs)
	}
	long := hoverActionsFor(row, []string{
		"paste", "pastePath", "copy", "reveal", "note", "star", "pinItem", "delete",
	})
	if len(long) != maxHoverActions {
		t.Errorf("len = %d", len(long))
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
