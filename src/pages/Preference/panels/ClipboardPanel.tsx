import { NumberField, Separator } from "@heroui/react";
import { useSnapshot } from "valtio";
import { settingsState, updateSettings } from "@/stores/settings";
import type {
  AutoPaste,
  Content,
  Feedback,
  History,
  RetentionUnit,
  Search,
  SearchPosition,
  Window,
  WindowPosition,
  WindowStyle,
} from "@/types/settings";
import { SelectControl, Toggle } from "../components/Field";
import Row from "../components/Row";

const patch = {
  content: (p: Partial<Content>) =>
    updateSettings({ clipboard: { content: p } }),
  feedback: (p: Partial<Feedback>) =>
    updateSettings({ clipboard: { feedback: p } }),
  history: (p: Partial<History>) =>
    updateSettings({ clipboard: { history: p } }),
  search: (p: Partial<Search>) => updateSettings({ clipboard: { search: p } }),
  window: (p: Partial<Window>) => updateSettings({ clipboard: { window: p } }),
};

const Section = ({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) => (
  <section>
    <h3 className="mb-1 font-medium text-default-600 text-xs uppercase tracking-wide">
      {title}
    </h3>
    <div className="flex flex-col divide-y divide-default-100">{children}</div>
  </section>
);

const ClipboardPanel = () => {
  const { value } = useSnapshot(settingsState);
  if (!value) return null;
  const { content, history, search, window, feedback } = value.clipboard;

  return (
    <div className="flex flex-col gap-4">
      <Section title="内容">
        <Row
          control={
            <SelectControl<AutoPaste>
              onChange={(v) => patch.content({ autoPaste: v })}
              options={[
                { key: "disabled", label: "仅选中" },
                { key: "singleClick", label: "单击粘贴" },
                { key: "doubleClick", label: "双击粘贴" },
              ]}
              value={content.autoPaste}
            />
          }
          label="点击行为"
        />
        <Row
          control={
            <Toggle
              isSelected={content.copyPlain}
              onChange={(v) => patch.content({ copyPlain: v })}
            />
          }
          label="复制时去除格式"
        />
        <Row
          control={
            <Toggle
              isSelected={content.pastePlain}
              onChange={(v) => patch.content({ pastePlain: v })}
            />
          }
          label="粘贴时去除格式"
        />
        <Row
          control={
            <Toggle
              isSelected={content.showOriginalPreview}
              onChange={(v) => patch.content({ showOriginalPreview: v })}
            />
          }
          description="悬停时显示 HTML/RTF 渲染前的原文"
          label="原始内容预览"
        />
        <Row
          control={
            <Toggle
              isSelected={content.deleteConfirm}
              onChange={(v) => patch.content({ deleteConfirm: v })}
            />
          }
          label="删除时二次确认"
        />
        <Row
          control={
            <Toggle
              isSelected={content.autoFavorite}
              onChange={(v) => patch.content({ autoFavorite: v })}
            />
          }
          label="自动收藏"
        />
        <Row
          control={
            <Toggle
              isSelected={content.autoSortByFrequency}
              onChange={(v) => patch.content({ autoSortByFrequency: v })}
            />
          }
          label="按使用频率排序"
        />
      </Section>

      <Separator />

      <Section title="历史">
        <Row
          control={
            <SelectControl<RetentionUnit>
              onChange={(v) =>
                patch.history({ retention: { ...history.retention, unit: v } })
              }
              options={[
                { key: "forever", label: "永久" },
                { key: "hours", label: "小时" },
                { key: "days", label: "天" },
                { key: "weeks", label: "周" },
                { key: "months", label: "月" },
              ]}
              value={history.retention.unit}
            />
          }
          description="超过指定时长的条目会被后台清理（收藏项除外）"
          label="保留时长"
        />
        {history.retention.unit !== "forever" && (
          <Row
            control={
              <NumberField
                className="w-28"
                minValue={1}
                onChange={(v) =>
                  patch.history({
                    retention: { ...history.retention, value: v || 1 },
                  })
                }
                value={history.retention.value || 1}
              >
                <NumberField.Group>
                  <NumberField.DecrementButton />
                  <NumberField.Input />
                  <NumberField.IncrementButton />
                </NumberField.Group>
              </NumberField>
            }
            label="时长数值"
          />
        )}
        <Row
          control={
            <NumberField
              className="w-28"
              minValue={0}
              onChange={(v) =>
                patch.history({ maxCount: Number.isFinite(v) ? v : 0 })
              }
              value={history.maxCount}
            >
              <NumberField.Group>
                <NumberField.DecrementButton />
                <NumberField.Input />
                <NumberField.IncrementButton />
              </NumberField.Group>
            </NumberField>
          }
          description="0 表示不限"
          label="最大条数"
        />
      </Section>

      <Separator />

      <Section title="搜索">
        <Row
          control={
            <SelectControl<SearchPosition>
              onChange={(v) => patch.search({ position: v })}
              options={[
                { key: "top", label: "顶部" },
                { key: "bottom", label: "底部" },
              ]}
              value={search.position}
            />
          }
          label="搜索框位置"
        />
        <Row
          control={
            <Toggle
              isSelected={search.defaultFocus}
              onChange={(v) => patch.search({ defaultFocus: v })}
            />
          }
          label="显示主窗时自动聚焦"
        />
        <Row
          control={
            <Toggle
              isSelected={search.clearOnHide}
              onChange={(v) => patch.search({ clearOnHide: v })}
            />
          }
          label="隐藏主窗时清空关键词"
        />
      </Section>

      <Separator />

      <Section title="窗口">
        <Row
          control={
            <SelectControl<WindowStyle>
              onChange={(v) => patch.window({ style: v })}
              options={[
                { key: "standard", label: "标准" },
                { key: "dock", label: "Dock" },
              ]}
              value={window.style}
            />
          }
          label="窗口样式"
        />
        <Row
          control={
            <SelectControl<WindowPosition>
              onChange={(v) => patch.window({ position: v })}
              options={[
                { key: "followCursor", label: "跟随光标" },
                { key: "center", label: "屏幕居中" },
                { key: "remember", label: "记住上次位置" },
              ]}
              value={window.position}
            />
          }
          label="弹出位置"
        />
        <Row
          control={
            <Toggle
              isSelected={window.alwaysOnTop}
              onChange={(v) => patch.window({ alwaysOnTop: v })}
            />
          }
          label="窗口置顶"
        />
        <Row
          control={
            <Toggle
              isSelected={window.allWorkspaces}
              onChange={(v) => patch.window({ allWorkspaces: v })}
            />
          }
          description="macOS Spaces / Windows 虚拟桌面"
          label="所有桌面可见"
        />
      </Section>

      <Separator />

      <Section title="反馈">
        <Row
          control={
            <Toggle
              isSelected={feedback.copySound}
              onChange={(v) => patch.feedback({ copySound: v })}
            />
          }
          label="复制成功提示音"
        />
      </Section>
    </div>
  );
};

export default ClipboardPanel;
