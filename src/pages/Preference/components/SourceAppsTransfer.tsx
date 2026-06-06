import { useMount } from "ahooks";
import { Button, Empty, Input, Popover, Transfer } from "antd";
import type { TransferKey } from "antd/es/transfer/interface";
import type { TFunction } from "i18next";
import type { ChangeEvent, FC } from "react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { listAllApps, refreshApps } from "@/commands";
import AssetImage from "@/components/AssetImage";
import Tooltip from "@/components/Tooltip";
import type { ClipboardApp } from "@/types/clipboard";
import type { Settings } from "@/types/settings";
import { log } from "@/utils/log";
import type {
  PreferenceSetting,
  PreferenceSettingChangeHandler,
} from "../types/preferences";
import PreferenceCountTag from "./PreferenceCountTag";

interface SourceAppsTransferProps {
  excludedAppsSetting: PreferenceSetting;
  scanDirsSetting: PreferenceSetting;
  settings: Settings;
  onChange: PreferenceSettingChangeHandler;
}

interface SourceAppTransferItem {
  description: string;
  iconPath: string | null;
  key: string;
  title: string;
}

/**
 * 来源应用复合设置：左侧展示全部应用，右侧维护已忽略应用。
 */
const SourceAppsTransfer: FC<SourceAppsTransferProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { excludedAppsSetting, scanDirsSetting, settings, onChange } = props;
  const excludedAppIds = settings.clipboard.filters.excludedAppIds;
  const scanDirs = settings.clipboard.filters.scanDirs;
  const [apps, setApps] = useState<ClipboardApp[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [scanDirsDraft, setScanDirsDraft] = useState(scanDirs.join("\n"));

  const dataSource = useMemo(() => {
    return buildTransferItems(apps, excludedAppIds);
  }, [apps, excludedAppIds]);

  const targetKeys = useMemo(() => {
    const knownKeys = new Set(
      dataSource.map((item) => {
        return item.key;
      }),
    );

    return excludedAppIds.filter((id) => {
      return knownKeys.has(id);
    });
  }, [dataSource, excludedAppIds]);

  /**
   * 拉取全部已知应用，并保留当前忽略列表中的占位项。
   */
  const loadApps = async () => {
    try {
      setApps(await listAllApps());
    } catch (error) {
      log.warn("load source apps failed", error);
    }
  };

  /**
   * 手动触发 Rust 侧目录扫描，完成后刷新穿梭框数据。
   */
  const handleRefreshApps = async () => {
    setRefreshing(true);

    try {
      setApps(await refreshApps());
    } catch (error) {
      log.warn("refresh source apps failed", error);
    } finally {
      setRefreshing(false);
    }
  };

  /**
   * 根据穿梭框右侧 key 列表保存忽略应用。
   */
  const handleTransferChange = async (nextKeys: TransferKey[]) => {
    await onChange(
      excludedAppsSetting,
      nextKeys.map((key) => {
        return String(key);
      }),
    );
  };

  const handleScanDirsDraftChange = (
    event: ChangeEvent<HTMLTextAreaElement>,
  ) => {
    setScanDirsDraft(event.target.value);
  };

  /**
   * 关闭目录设置浮层时，把草稿重置回最新设置值。
   */
  const handlePopoverOpenChange = (nextOpen: boolean) => {
    setPopoverOpen(nextOpen);

    if (nextOpen) return;

    setScanDirsDraft(scanDirs.join("\n"));
  };

  /**
   * 保存应用发现目录；路径变更后主动扫描一次，保证左侧列表及时更新。
   */
  const handleSaveScanDirs = async () => {
    const nextScanDirs = normalizeScanDirs(scanDirsDraft);

    await onChange(scanDirsSetting, nextScanDirs);
    setPopoverOpen(false);

    if (arrayEqual(scanDirs, nextScanDirs)) return;

    await handleRefreshApps();
  };

  /**
   * 取消目录设置编辑并关闭浮层。
   */
  const handleCancelScanDirs = () => {
    setScanDirsDraft(scanDirs.join("\n"));
    setPopoverOpen(false);
  };

  useMount(() => {
    void loadApps();
  });

  const sourceCountLabel = formatCountLabel(t, dataSource.length);
  const targetCountLabel = formatCountLabel(t, targetKeys.length);
  const sourceTitle = (
    <div className="flex min-w-0 items-center justify-between gap-2">
      <span className="flex min-w-0 items-center gap-1">
        <span className="truncate">
          {t("schema.settings.source.appTransfer.allTitle")}
        </span>
        <PreferenceCountTag>{sourceCountLabel}</PreferenceCountTag>
      </span>
      <div className="flex shrink-0 items-center gap-1">
        <Popover
          content={
            <ScanDirsPopoverContent
              draft={scanDirsDraft}
              onCancel={handleCancelScanDirs}
              onDraftChange={handleScanDirsDraftChange}
              onSave={handleSaveScanDirs}
            />
          }
          onOpenChange={handlePopoverOpenChange}
          open={popoverOpen}
          placement="bottomRight"
          trigger="click"
        >
          <Tooltip title={t("schema.settings.source.appTransfer.scanDirs")}>
            <Button
              aria-label={t("schema.settings.source.appTransfer.scanDirs")}
              icon={<i aria-hidden="true" className="i-lucide:folder-search" />}
              size="small"
              type="text"
            />
          </Tooltip>
        </Popover>

        <Tooltip title={t("schema.settings.source.appTransfer.refreshApps")}>
          <Button
            aria-label={t("schema.settings.source.appTransfer.refreshApps")}
            icon={<i aria-hidden="true" className="i-lucide:refresh-cw" />}
            loading={refreshing}
            onClick={handleRefreshApps}
            size="small"
            type="text"
          />
        </Tooltip>
      </div>
    </div>
  );

  const targetTitle = (
    <span className="flex min-w-0 items-center gap-1">
      <span className="truncate">
        {t("schema.settings.source.appTransfer.ignoredTitle")}
      </span>
      <PreferenceCountTag>{targetCountLabel}</PreferenceCountTag>
    </span>
  );

  return (
    <div
      className="flex h-full min-h-0 flex-col"
      data-preference-setting-id={excludedAppsSetting.id}
    >
      <Transfer<SourceAppTransferItem>
        className="source-apps-transfer h-full min-h-0"
        classNames={{
          body: "min-h-0 flex-1",
          header: "h-11 [&_.ant-transfer-list-header-selected]:hidden",
          item: "h-10",
          itemContent: "min-w-0",
          list: "h-full",
          root: "h-full min-h-0 w-full",
          section: "h-full! min-w-0 flex-1",
          title: "ms-0! min-w-0 flex-1! text-start!",
        }}
        dataSource={dataSource}
        filterOption={filterTransferItem}
        locale={{
          itemsUnit: t("schema.settings.source.appTransfer.itemsUnit"),
          itemUnit: t("schema.settings.source.appTransfer.itemUnit"),
          notFoundContent: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />,
          searchPlaceholder: t(
            "schema.settings.source.appTransfer.searchPlaceholder",
          ),
        }}
        onChange={handleTransferChange}
        render={renderTransferItem}
        showSearch
        showSelectAll={false}
        targetKeys={targetKeys}
        titles={[sourceTitle, targetTitle]}
      />
    </div>
  );
};

export default SourceAppsTransfer;

interface ScanDirsPopoverContentProps {
  draft: string;
  onCancel: () => void;
  onDraftChange: (event: ChangeEvent<HTMLTextAreaElement>) => void;
  onSave: () => Promise<void>;
}

/**
 * 应用发现目录编辑浮层，保存后由父组件提交设置并刷新列表。
 */
const ScanDirsPopoverContent: FC<ScanDirsPopoverContentProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { draft, onCancel, onDraftChange, onSave } = props;

  return (
    <div className="w-80">
      <Input.TextArea
        autoSize={{ maxRows: 8, minRows: 5 }}
        onChange={onDraftChange}
        placeholder={t("schema.settings.source.scanDirs.placeholder")}
        value={draft}
      />
      <div className="mt-3 flex justify-end gap-2">
        <Button onClick={onCancel} size="small">
          {t("schema.settings.source.appTransfer.cancel")}
        </Button>
        <Button onClick={onSave} size="small" type="primary">
          {t("schema.settings.source.appTransfer.save")}
        </Button>
      </div>
    </div>
  );
};

/**
 * 把 Rust 返回的应用列表与当前忽略列表合并成 Transfer 数据。
 */
function buildTransferItems(
  apps: ClipboardApp[],
  excludedAppIds: string[],
): SourceAppTransferItem[] {
  const items = new Map(
    apps.map((app) => {
      return [
        app.id,
        {
          description: app.id,
          iconPath: app.iconPath,
          key: app.id,
          title: app.name,
        },
      ];
    }),
  );

  for (const id of excludedAppIds) {
    if (items.has(id)) continue;

    items.set(id, {
      description: id,
      iconPath: null,
      key: id,
      title: id,
    });
  }

  return Array.from(items.values()).sort((left, right) => {
    return left.title
      .toLocaleLowerCase()
      .localeCompare(right.title.toLocaleLowerCase());
  });
}

/**
 * 搜索应用名和 bundle id / 可执行路径。
 */
function filterTransferItem(inputValue: string, item: SourceAppTransferItem) {
  const query = inputValue.trim().toLocaleLowerCase();
  if (!query) return true;

  return `${item.title}\n${item.description}`
    .toLocaleLowerCase()
    .includes(query);
}

/**
 * 渲染应用条目，只展示对用户可识别的应用名。
 */
function renderTransferItem(item: SourceAppTransferItem) {
  return {
    label: (
      <div className="flex min-w-0 items-center gap-2">
        <span className="flex size-7 shrink-0 items-center justify-center overflow-hidden rounded-1.5 bg-ant-fill-quaternary text-ant-secondary">
          {item.iconPath ? (
            <AssetImage
              alt={item.title}
              className="size-7"
              src={item.iconPath}
            />
          ) : (
            <i aria-hidden="true" className="i-lucide:app-window text-base" />
          )}
        </span>
        <span className="min-w-0">
          <span className="block truncate font-medium text-sm">
            {item.title}
          </span>
        </span>
      </div>
    ),
    value: item.title,
  };
}

/**
 * 格式化标题里的数量标签。
 */
function formatCountLabel(t: TFunction<"preferences">, count: number) {
  const unitKey =
    count === 1
      ? "schema.settings.source.appTransfer.itemUnit"
      : "schema.settings.source.appTransfer.itemsUnit";

  return `${count} ${t(unitKey)}`;
}

/**
 * 把目录输入归一成非空、去重、保持顺序的数组。
 */
function normalizeScanDirs(draft: string) {
  const seen = new Set<string>();

  return draft
    .split("\n")
    .map((line) => {
      return line.trim();
    })
    .filter((line) => {
      if (line.length === 0) return false;
      if (seen.has(line)) return false;

      seen.add(line);
      return true;
    });
}

/**
 * 比较字符串数组是否完全一致。
 */
function arrayEqual(left: string[], right: string[]) {
  if (left.length !== right.length) return false;

  return left.every((item, index) => {
    return item === right[index];
  });
}
