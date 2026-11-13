package database

import (
	"context"
	"testing"
)

func TestCreateGroupAppendsAndListsInOrder(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	first, err := db.CreateGroup(ctx, "Work", "i-lets-icons:folder")
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.CreateGroup(ctx, "Ideas", "i-lets-icons:folder")
	if err != nil {
		t.Fatal(err)
	}
	groups, err := db.ListGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups = %+v", groups)
	}
	if groups[0].ID != first.ID || groups[1].ID != second.ID {
		t.Fatalf("order = %q, %q", groups[0].ID, groups[1].ID)
	}
	if first.SortOrder != 0 || second.SortOrder != 1 {
		t.Errorf("sort order = %d, %d", first.SortOrder, second.SortOrder)
	}
}

func TestRenameGroupPreservesSortOrder(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	group, err := db.CreateGroup(ctx, "Old", "i-lets-icons:folder")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RenameGroup(ctx, group.ID, "New"); err != nil {
		t.Fatal(err)
	}
	found, err := db.FindGroup(ctx, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.Name != "New" || found.SortOrder != group.SortOrder {
		t.Fatalf("group = %+v", found)
	}
}

func TestDeleteGroupKeepsItemsAndClearsGroupID(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	group, err := db.CreateGroup(ctx, "Temporary", "i-lets-icons:folder")
	if err != nil {
		t.Fatal(err)
	}
	groupID := group.ID
	item := testItem(KindText, "keep me", "keep me")
	item.GroupID = &groupID
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	before, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.DeleteGroup(ctx, group.ID); err != nil {
		t.Fatal(err)
	}
	after, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if after == nil {
		t.Fatal("item was deleted with its group")
	}
	if after.GroupID != nil {
		t.Errorf("group id = %v, want nil", after.GroupID)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("updated_at moved: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestSetGroupDoesNotRefreshUpdatedAt(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	group, err := db.CreateGroup(ctx, "Work", "i-lets-icons:folder")
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := db.Upsert(ctx, testItem(KindText, "hello", "hello"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	groupID := group.ID
	if err := db.SetGroup(ctx, id, &groupID); err != nil {
		t.Fatal(err)
	}
	after, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if after.GroupID == nil || *after.GroupID != groupID {
		t.Errorf("group id = %v", after.GroupID)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("updated_at moved: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestPageFiltersByGroup(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	group, err := db.CreateGroup(ctx, "Work", "i-lets-icons:folder")
	if err != nil {
		t.Fatal(err)
	}
	groupID := group.ID
	inside := testItem(KindText, "inside", "inside")
	inside.GroupID = &groupID
	if _, _, err := db.Upsert(ctx, inside); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.Upsert(ctx, testItem(KindText, "outside", "outside")); err != nil {
		t.Fatal(err)
	}
	page, err := db.Page(ctx, Query{GroupID: &groupID})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Summary == nil || *page.Items[0].Summary != "inside" {
		t.Fatalf("page = %+v", page)
	}
}
