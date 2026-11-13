//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// buildInteraction adds mouse click behavior and paste defaults.
func (v *SettingsView) buildInteraction() {
	content := v.newTab("interaction", "交互")

	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString("左键和中键动作在剪贴板面板中生效；自动粘贴需要辅助功能权限。"))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 396.0
	v.addPopup(content, y, "鼠标左键", "content.autoPaste",
		[]settingsOption{
			{Title: "禁用", Value: "disabled"},
			{Title: "单击粘贴", Value: "singleClickPaste"},
			{Title: "双击粘贴", Value: "doubleClickPaste"},
			{Title: "单击复制", Value: "singleClickCopy"},
			{Title: "双击复制", Value: "doubleClickCopy"},
		},
		func(s SettingsValues) string { return s.AutoPaste })
	v.addPopup(content, y-52, "鼠标中键", "content.middleClick",
		[]settingsOption{
			{Title: "禁用", Value: "disabled"},
			{Title: "粘贴", Value: "singleClickPaste"},
			{Title: "粘贴为纯文本", Value: "singleClickPastePlain"},
			{Title: "复制", Value: "singleClickCopy"},
			{Title: "复制为纯文本", Value: "singleClickCopyPlain"},
		},
		func(s SettingsValues) string { return s.MiddleClick })
	v.addSwitch(content, y-104, "粘贴文件时写入路径", "content.pasteFilesAsPath",
		func(s SettingsValues) bool { return s.PasteFilesAsPath })
	v.addSwitch(content, y-142, "有备注时自动收藏", "content.autoFavorite",
		func(s SettingsValues) bool { return s.AutoFavorite })

	note := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString("“粘贴文件时写入路径”只影响文件记录；普通文本仍按原内容粘贴。"))
	note.Send(sel("setFont:"), systemFont(11))
	note.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	note.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 190}, Size: CGSize{Width: tabContentW - 48, Height: 32}})
	content.Send(sel("addSubview:"), note)
}

// buildDeletion adds the per-category delete protection rules.
func (v *SettingsView) buildDeletion() {
	content := v.newTab("deletion", "删除")

	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString("确认开关只影响对应类型；关闭“允许删除”后，该类型记录不再显示删除入口。"))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 396.0
	v.addSwitch(content, y, "普通条目删除前确认", "content.deleteConfirm",
		func(s SettingsValues) bool { return s.DeleteConfirm })
	v.addSwitch(content, y-38, "收藏条目删除前确认", "content.deleteFavoriteConfirm",
		func(s SettingsValues) bool { return s.DeleteFavoriteConfirm })
	v.addSwitch(content, y-76, "置顶条目删除前确认", "content.deletePinnedConfirm",
		func(s SettingsValues) bool { return s.DeletePinnedConfirm })
	v.addSwitch(content, y-120, "允许删除收藏条目", "content.deleteFavoriteItems",
		func(s SettingsValues) bool { return s.DeleteFavoriteItems })
	v.addSwitch(content, y-158, "允许删除置顶条目", "content.deletePinnedItems",
		func(s SettingsValues) bool { return s.DeletePinnedItems })
	v.addSwitch(content, y-196, "收藏项仅在收藏分组可删除", "content.deleteFavoriteOnlyFav",
		func(s SettingsValues) bool { return s.DeleteFavoriteOnlyFav })
}
