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
