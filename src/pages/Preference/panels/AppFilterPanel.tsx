import { Button, Checkbox, Input, Separator } from "@heroui/react";
import { convertFileSrc, invoke } from "@tauri-apps/api/core";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { TAURI_COMMAND } from "@/constants/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import { settingsState, updateSettings } from "@/stores/settings";
import type { ClipboardApp } from "@/types/clipboard";
import { log } from "@/utils/log";

const patchExcluded = (excludedAppIds: string[]) =>
  updateSettings({ clipboard: { filters: { excludedAppIds } } });

const patchDirs = (scanDirs: string[]) =>
  updateSettings({ clipboard: { filters: { scanDirs } } });

const AppFilterPanel = () => {
  const { t } = useTranslation();
  const { value } = useSnapshot(settingsState);
  const [apps, setApps] = useState<ClipboardApp[]>([]);
  const [keyword, setKeyword] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const reload = useCallback(() => {
    invoke<ClipboardApp[]>(TAURI_COMMAND.LIST_ALL_APPS)
      .then(setApps)
      .catch((err) => log.error("list_all_apps failed", err));
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  // 后台扫描完成 / 图标分批入库后 Rust 会 emit，重拉以增量刷新。
  useTauriListen(TAURI_EVENT.CLIPBOARD_APPS_UPDATED, reload);

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase();
    if (!k) return apps;
    return apps.filter(
      (a) => a.name.toLowerCase().includes(k) || a.id.toLowerCase().includes(k),
    );
  }, [apps, keyword]);

  if (!value) return null;
  const { excludedAppIds, scanDirs } = value.clipboard.filters;
  const excludedSet = new Set(excludedAppIds);
  const excludedApps = apps.filter((a) => excludedSet.has(a.id));

  const toggleExcluded = (id: string, checked: boolean) => {
    const next = checked
      ? Array.from(new Set([...excludedAppIds, id]))
      : excludedAppIds.filter((x) => x !== id);
    void patchExcluded(next);
  };

  const refresh = async () => {
    setRefreshing(true);
    try {
      const next = await invoke<ClipboardApp[]>(TAURI_COMMAND.REFRESH_APPS);
      setApps(next);
    } catch (err) {
      log.error("refresh_apps failed", err);
    } finally {
      setRefreshing(false);
    }
  };

  const updateDir = (idx: number, dir: string) => {
    const next = [...scanDirs];
    next[idx] = dir;
    void patchDirs(next);
  };

  const addDir = () => {
    void patchDirs([...scanDirs, ""]);
  };

  const removeDir = (idx: number) => {
    void patchDirs(scanDirs.filter((_, i) => i !== idx));
  };

  return (
    <div className="flex flex-col gap-4">
      <section className="flex flex-col gap-2">
        <header className="flex items-center justify-between">
          <h3 className="font-medium text-default-600 text-xs uppercase tracking-wide">
            {t("filters.section.scanDirs")}
          </h3>
          <Button onPress={addDir} size="sm" variant="ghost">
            {t("filters.dirs.add")}
          </Button>
        </header>
        <p className="text-default-500 text-xs">
          {t("filters.section.scanDirs.desc")}
        </p>
        <div className="flex flex-col gap-2">
          {scanDirs.length === 0 && (
            <div className="text-default-400 text-xs">
              {t("filters.dirs.empty")}
            </div>
          )}
          {scanDirs.map((dir, idx) => (
            <DirRow
              dir={dir}
              // biome-ignore lint/suspicious/noArrayIndexKey: 目录行通过 idx 配合输入框 defaultValue 控制
              key={`dir-${idx}`}
              onChange={(v) => updateDir(idx, v)}
              onRemove={() => removeDir(idx)}
              removeLabel={t("filters.dirs.remove")}
            />
          ))}
        </div>
      </section>

      <Separator />

      <section className="flex flex-col gap-2">
        <header className="flex items-center justify-between gap-2">
          <h3 className="font-medium text-default-600 text-xs uppercase tracking-wide">
            {t("filters.section.excluded")}
          </h3>
          <span className="text-default-400 text-xs">
            {excludedApps.length}
          </span>
        </header>
        <p className="text-default-500 text-xs">
          {t("filters.section.excluded.desc")}
        </p>
        <div className="flex max-h-[30vh] flex-col divide-y divide-default-100 overflow-auto rounded border border-default-200">
          {excludedApps.length === 0 && (
            <div className="p-3 text-default-400 text-xs">
              {t("filters.excluded.empty")}
            </div>
          )}
          {excludedApps.map((app) => (
            <AppRow
              app={app}
              checked
              key={`excluded-${app.id}`}
              onToggle={(c) => toggleExcluded(app.id, c)}
            />
          ))}
        </div>
      </section>

      <Separator />

      <section className="flex min-h-0 flex-1 flex-col gap-2">
        <header className="flex items-center justify-between gap-2">
          <h3 className="font-medium text-default-600 text-xs uppercase tracking-wide">
            {t("filters.section.apps")}
          </h3>
          <Button
            isDisabled={refreshing}
            onPress={refresh}
            size="sm"
            variant="ghost"
          >
            {t("filters.apps.refresh")}
          </Button>
        </header>
        <p className="text-default-500 text-xs">
          {t("filters.section.apps.desc")}
        </p>
        <Input
          onChange={(e) => setKeyword(e.target.value)}
          placeholder={t("filters.apps.searchPlaceholder")}
          value={keyword}
        />
        <div className="flex max-h-[55vh] flex-col divide-y divide-default-100 overflow-auto rounded border border-default-200">
          {filtered.length === 0 && (
            <div className="p-3 text-default-400 text-xs">
              {t("filters.apps.empty")}
            </div>
          )}
          {filtered.map((app) => (
            <AppRow
              app={app}
              checked={excludedSet.has(app.id)}
              key={app.id}
              onToggle={(c) => toggleExcluded(app.id, c)}
            />
          ))}
        </div>
      </section>
    </div>
  );
};

const DirRow = ({
  dir,
  onChange,
  onRemove,
  removeLabel,
}: {
  dir: string;
  onChange: (v: string) => void;
  onRemove: () => void;
  removeLabel: string;
}) => {
  const [draft, setDraft] = useState(dir);
  useEffect(() => setDraft(dir), [dir]);
  return (
    <div className="flex items-center gap-2">
      <Input
        className="flex-1"
        onBlur={() => {
          if (draft !== dir) onChange(draft);
        }}
        onChange={(e) => setDraft(e.target.value)}
        value={draft}
      />
      <Button onPress={onRemove} size="sm" variant="ghost">
        {removeLabel}
      </Button>
    </div>
  );
};

const AppRow = ({
  app,
  checked,
  onToggle,
}: {
  app: ClipboardApp;
  checked: boolean;
  onToggle: (c: boolean) => void;
}) => {
  const [iconSrc, setIconSrc] = useState<string | null>(null);
  useEffect(() => {
    if (!app.iconFile) {
      setIconSrc(null);
      return;
    }
    invoke<string>(TAURI_COMMAND.GET_CLIPBOARD_APP_ICON_PATH, {
      fileName: app.iconFile,
    })
      .then((p) => setIconSrc(convertFileSrc(p)))
      .catch(() => setIconSrc(null));
  }, [app.iconFile]);

  return (
    <button
      className="flex w-full cursor-pointer items-center gap-3 px-3 py-2 text-left hover:bg-default-50"
      onClick={() => onToggle(!checked)}
      type="button"
    >
      <Checkbox isSelected={checked} onChange={onToggle} />
      <div className="size-6 shrink-0 overflow-hidden rounded">
        {iconSrc ? (
          <img
            alt=""
            className="size-full object-contain"
            loading="lazy"
            src={iconSrc}
          />
        ) : (
          <div className="size-full bg-default-100" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm">{app.name}</div>
        <div className="truncate text-default-400 text-xs">{app.id}</div>
      </div>
    </button>
  );
};

export default AppFilterPanel;
