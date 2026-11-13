package app

import (
	"context"
	"strings"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

func TestCreateGroupNormalizesAndSelects(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	editGroup = func(string) (string, bool) { return "  Work  ", true }
	t.Cleanup(func() { editGroup = macos.EditGroupName })

	option, created, err := model.CreateGroup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !created || option == nil || option.Name != "Work" {
		t.Fatalf("created = %v, option = %+v", created, option)
	}
	group, err := db.FindGroup(ctx, option.ID)
	if err != nil {
		t.Fatal(err)
	}
	if group == nil || group.Icon != defaultGroupIcon {
		t.Fatalf("group = %+v", group)
	}
}

func TestCreateGroupRejectsEmptyLongAndDuplicateNames(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	if _, err := db.CreateGroup(ctx, "Work", defaultGroupIcon); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { editGroup = macos.EditGroupName })

	cases := []struct {
		value string
		want  string
	}{
		{"   ", "不能为空"},
		{strings.Repeat("a", groupNameMaxChars+1), "不能超过"},
		{"work", duplicateGroupError},
	}
	for _, tc := range cases {
		editGroup = func(string) (string, bool) { return tc.value, true }
		if _, created, err := model.CreateGroup(ctx); created || err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: created = %v, err = %v", tc.value, created, err)
		}
	}
}

func TestRenameGroupCancelAndDuplicate(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	work, err := db.CreateGroup(ctx, "Work", defaultGroupIcon)
	if err != nil {
		t.Fatal(err)
	}
	ideas, err := db.CreateGroup(ctx, "Ideas", defaultGroupIcon)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { editGroup = macos.EditGroupName })

	editGroup = func(string) (string, bool) { return "", false }
	renamed, err := model.RenameGroup(ctx, work.ID)
	if err != nil || renamed {
		t.Fatalf("cancel: renamed = %v, err = %v", renamed, err)
	}

	editGroup = func(string) (string, bool) { return "ideas", true }
	renamed, err = model.RenameGroup(ctx, work.ID)
	if err == nil || renamed || !strings.Contains(err.Error(), duplicateGroupError) {
		t.Fatalf("duplicate: renamed = %v, err = %v", renamed, err)
	}

	editGroup = func(string) (string, bool) { return "Projects", true }
	renamed, err = model.RenameGroup(ctx, work.ID)
	if err != nil || !renamed {
		t.Fatalf("rename: renamed = %v, err = %v", renamed, err)
	}
	found, err := db.FindGroup(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Name != "Projects" || ideas.ID == "" {
		t.Fatalf("group = %+v", found)
	}
}

func TestDeleteGroupRequiresConfirmationAndKeepsItems(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	group, err := db.CreateGroup(ctx, "Temporary", defaultGroupIcon)
	if err != nil {
		t.Fatal(err)
	}
	groupID := group.ID
	item := &database.Item{
		Kind: database.KindText, Content: "hello", GroupID: &groupID, Platform: "macos",
	}
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { deleteGroup = macos.ConfirmDeleteGroup })

	deleteGroup = func(string) bool { return false }
	deleted, err := model.DeleteGroup(ctx, group.ID)
	if err != nil || deleted {
		t.Fatalf("cancel: deleted = %v, err = %v", deleted, err)
	}
	if found, err := db.FindGroup(ctx, group.ID); err != nil || found == nil {
		t.Fatalf("group after cancel = %+v, err = %v", found, err)
	}

	deleteGroup = func(string) bool { return true }
	deleted, err = model.DeleteGroup(ctx, group.ID)
	if err != nil || !deleted {
		t.Fatalf("delete: deleted = %v, err = %v", deleted, err)
	}
	found, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.GroupID != nil {
		t.Fatalf("item after delete = %+v", found)
	}
}

func TestMoveToGroupAndBackToUngrouped(t *testing.T) {
	model, db := actionModel(t)
	ctx := context.Background()
	group, err := db.CreateGroup(ctx, "Work", defaultGroupIcon)
	if err != nil {
		t.Fatal(err)
	}
	item := &database.Item{Kind: database.KindText, Content: "hello", Platform: "macos"}
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.MoveToGroup(ctx, id, group.ID); err != nil {
		t.Fatal(err)
	}
	found, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found.GroupID == nil || *found.GroupID != group.ID {
		t.Fatalf("group id = %v", found.GroupID)
	}
	if err := model.MoveToGroup(ctx, id, ""); err != nil {
		t.Fatal(err)
	}
	found, err = db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found.GroupID != nil {
		t.Fatalf("group id = %v", found.GroupID)
	}
	if err := model.MoveToGroup(ctx, id, "missing"); err == nil {
		t.Fatal("missing group was accepted")
	}
}
