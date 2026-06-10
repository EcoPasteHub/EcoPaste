# TODO: 列表条目显示上限

## 目标

让用户在设置里调整列表密度：文字最长显示行数、图片最大高度、文件最多显示数量。

## 设置模型

在 `Settings.clipboard.content` 中新增：

- `textMaxLines: number`，默认 3，范围 1-10，0 表示不限。
- `imageMaxHeight: number`，默认 80，范围 40-240，步进 8。
- `filesMaxVisible: number`，默认 1，范围 1-10。

这些字段只是展示偏好，不影响捕获、存储或同步。

## 前端渲染

- `TextCard` 使用动态 clamp；不限时移除 clamp。
- `ImageCard` 使用设置驱动的最大高度，保持 `object-contain`。
- `FilesCard` 渲染前 N 个文件，超出时显示 `+N`。
- 避免每行单独订阅全量设置；在列表上层读取设置后作为 props 传入。

## 设置面板

- 偏好设置的剪贴板面板新增“列表显示”分组。
- 使用 Ant Design `Slider`、`InputNumber` 或现有 setting control。
- 调整后实时应用到主列表。
- i18n 补齐中英文文案。

## 注意

- 项目约定不写任意 px 样式；这里的高度来自用户设置，是动态运行值，若用 inline style 需要局部说明并保持范围约束。
- Virtuoso 动态行高可能抖动，极端值下需手动验证滚动稳定性。

## 验收

- 三个设置项即时生效。
- 极端值下列表不重叠、不遮挡、不撑破主窗口。
- 重启后设置持久化。
- macOS 与 Windows 主窗口行为一致。

