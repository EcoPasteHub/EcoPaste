import { Divider, InputNumber } from "antd";
import { useTranslation } from "react-i18next";
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
    <h3 className="c-text-secondary mb-1 font-medium text-xs uppercase tracking-wide">
      {title}
    </h3>
    <div className="flex flex-col divide-y divide-border-secondary">
      {children}
    </div>
  </section>
);

const AUTO_PASTE_KEYS: AutoPaste[] = ["disabled", "singleClick", "doubleClick"];
const RETENTION_UNITS: RetentionUnit[] = [
  "forever",
  "hours",
  "days",
  "weeks",
  "months",
];
const SEARCH_POSITIONS: SearchPosition[] = ["top", "bottom"];
const WINDOW_STYLES: WindowStyle[] = ["standard", "dock"];
const WINDOW_POSITIONS: WindowPosition[] = [
  "followCursor",
  "center",
  "remember",
];

const ClipboardPanel = () => {
  const { t } = useTranslation();
  const { value } = useSnapshot(settingsState);
  if (!value) return null;
  const { content, history, search, window, feedback } = value.clipboard;

  return (
    <div className="flex flex-col gap-4">
      <Section title={t("clipboard.section.content")}>
        <Row
          control={
            <SelectControl<AutoPaste>
              onChange={(v) => patch.content({ autoPaste: v })}
              options={AUTO_PASTE_KEYS.map((key) => ({
                key,
                label: t(`clipboard.autoPaste.option.${key}`),
              }))}
              value={content.autoPaste}
            />
          }
          label={t("clipboard.autoPaste.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={content.copyPlain}
              onChange={(v) => patch.content({ copyPlain: v })}
            />
          }
          label={t("clipboard.copyPlain.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={content.pastePlain}
              onChange={(v) => patch.content({ pastePlain: v })}
            />
          }
          label={t("clipboard.pastePlain.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={content.showOriginalPreview}
              onChange={(v) => patch.content({ showOriginalPreview: v })}
            />
          }
          description={t("clipboard.showOriginalPreview.desc")}
          label={t("clipboard.showOriginalPreview.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={content.deleteConfirm}
              onChange={(v) => patch.content({ deleteConfirm: v })}
            />
          }
          label={t("clipboard.deleteConfirm.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={content.autoFavorite}
              onChange={(v) => patch.content({ autoFavorite: v })}
            />
          }
          label={t("clipboard.autoFavorite.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={content.autoSortByFrequency}
              onChange={(v) => patch.content({ autoSortByFrequency: v })}
            />
          }
          label={t("clipboard.autoSortByFrequency.label")}
        />
      </Section>

      <Divider />

      <Section title={t("clipboard.section.history")}>
        <Row
          control={
            <SelectControl<RetentionUnit>
              onChange={(v) =>
                patch.history({ retention: { ...history.retention, unit: v } })
              }
              options={RETENTION_UNITS.map((key) => ({
                key,
                label: t(`clipboard.retention.unit.${key}`),
              }))}
              value={history.retention.unit}
            />
          }
          description={t("clipboard.retention.desc")}
          label={t("clipboard.retention.label")}
        />
        {history.retention.unit !== "forever" && (
          <Row
            control={
              <InputNumber
                className="w-28"
                min={1}
                onChange={(v) =>
                  patch.history({
                    retention: {
                      ...history.retention,
                      value: Number(v) || 1,
                    },
                  })
                }
                value={history.retention.value || 1}
              />
            }
            label={t("clipboard.retentionValue.label")}
          />
        )}
        <Row
          control={
            <InputNumber
              className="w-28"
              min={0}
              onChange={(v) => {
                const n = Number(v);
                patch.history({ maxCount: Number.isFinite(n) ? n : 0 });
              }}
              value={history.maxCount}
            />
          }
          description={t("clipboard.maxCount.desc")}
          label={t("clipboard.maxCount.label")}
        />
      </Section>

      <Divider />

      <Section title={t("clipboard.section.search")}>
        <Row
          control={
            <SelectControl<SearchPosition>
              onChange={(v) => patch.search({ position: v })}
              options={SEARCH_POSITIONS.map((key) => ({
                key,
                label: t(`clipboard.searchPosition.option.${key}`),
              }))}
              value={search.position}
            />
          }
          label={t("clipboard.searchPosition.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={search.defaultFocus}
              onChange={(v) => patch.search({ defaultFocus: v })}
            />
          }
          label={t("clipboard.defaultFocus.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={search.clearOnHide}
              onChange={(v) => patch.search({ clearOnHide: v })}
            />
          }
          label={t("clipboard.clearOnHide.label")}
        />
      </Section>

      <Divider />

      <Section title={t("clipboard.section.window")}>
        <Row
          control={
            <SelectControl<WindowStyle>
              onChange={(v) => patch.window({ style: v })}
              options={WINDOW_STYLES.map((key) => ({
                key,
                label: t(`clipboard.windowStyle.option.${key}`),
              }))}
              value={window.style}
            />
          }
          label={t("clipboard.windowStyle.label")}
        />
        <Row
          control={
            <SelectControl<WindowPosition>
              onChange={(v) => patch.window({ position: v })}
              options={WINDOW_POSITIONS.map((key) => ({
                key,
                label: t(`clipboard.windowPosition.option.${key}`),
              }))}
              value={window.position}
            />
          }
          label={t("clipboard.windowPosition.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={window.alwaysOnTop}
              onChange={(v) => patch.window({ alwaysOnTop: v })}
            />
          }
          label={t("clipboard.alwaysOnTop.label")}
        />
        <Row
          control={
            <Toggle
              isSelected={window.allWorkspaces}
              onChange={(v) => patch.window({ allWorkspaces: v })}
            />
          }
          description={t("clipboard.allWorkspaces.desc")}
          label={t("clipboard.allWorkspaces.label")}
        />
      </Section>

      <Divider />

      <Section title={t("clipboard.section.feedback")}>
        <Row
          control={
            <Toggle
              isSelected={feedback.copySound}
              onChange={(v) => patch.feedback({ copySound: v })}
            />
          }
          label={t("clipboard.copySound.label")}
        />
      </Section>
    </div>
  );
};

export default ClipboardPanel;
