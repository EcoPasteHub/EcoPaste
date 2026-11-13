package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

var (
	editGroup   = macos.EditGroupName
	deleteGroup = macos.ConfirmDeleteGroup
)

const (
	groupNameMaxChars   = 32
	defaultGroupIcon    = "i-lets-icons:folder"
	duplicateGroupError = "已存在同名分组"
)

func (m *historyModel) Groups(ctx context.Context) ([]ui.GroupOption, error) {
	groups, err := m.runtime.db.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]ui.GroupOption, 0, len(groups))
	for _, group := range groups {
		options = append(options, groupOption(group))
	}
	return options, nil
}

func (m *historyModel) CreateGroup(ctx context.Context) (*ui.GroupOption, bool, error) {
	value, ok := editGroup("")
	if !ok {
		return nil, false, nil
	}
	name, err := normalizeGroupName(value)
	if err != nil {
		return nil, false, err
	}
	if err := m.ensureUniqueGroupName(ctx, name, ""); err != nil {
		return nil, false, err
	}
	group, err := m.runtime.db.CreateGroup(ctx, name, defaultGroupIcon)
	if err != nil {
		return nil, false, err
	}
	return &ui.GroupOption{ID: group.ID, Name: group.Name}, true, nil
}

func (m *historyModel) RenameGroup(ctx context.Context, id string) (bool, error) {
	group, err := m.runtime.db.FindGroup(ctx, id)
	if err != nil {
		return false, err
	}
	if group == nil {
		return false, fmt.Errorf("分组不存在")
	}
	value, ok := editGroup(group.Name)
	if !ok {
		return false, nil
	}
	name, err := normalizeGroupName(value)
	if err != nil {
		return false, err
	}
	if name == group.Name {
		return false, nil
	}
	if err := m.ensureUniqueGroupName(ctx, name, id); err != nil {
		return false, err
	}
	if err := m.runtime.db.RenameGroup(ctx, id, name); err != nil {
		return false, err
	}
	return true, nil
}

func (m *historyModel) DeleteGroup(ctx context.Context, id string) (bool, error) {
	group, err := m.runtime.db.FindGroup(ctx, id)
	if err != nil {
		return false, err
	}
	if group == nil {
		return false, fmt.Errorf("分组不存在")
	}
	if !deleteGroup(group.Name) {
		return false, nil
	}
	if err := m.runtime.db.DeleteGroup(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func (m *historyModel) MoveToGroup(ctx context.Context, itemID, groupID string) error {
	var value *string
	if groupID != "" {
		group, err := m.runtime.db.FindGroup(ctx, groupID)
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("分组不存在")
		}
		value = &groupID
	}
	return m.runtime.db.SetGroup(ctx, itemID, value)
}

func (m *historyModel) ensureUniqueGroupName(ctx context.Context, name, exceptID string) error {
	groups, err := m.runtime.db.ListGroups(ctx)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.ID != exceptID && strings.EqualFold(group.Name, name) {
			return fmt.Errorf("%s", duplicateGroupError)
		}
	}
	return nil
}

func normalizeGroupName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("分组名称不能为空")
	}
	if runes := []rune(name); len(runes) > groupNameMaxChars {
		return "", fmt.Errorf("分组名称不能超过 %d 个字符", groupNameMaxChars)
	}
	return name, nil
}

func groupOption(group database.Group) ui.GroupOption {
	return ui.GroupOption{ID: group.ID, Name: group.Name}
}
