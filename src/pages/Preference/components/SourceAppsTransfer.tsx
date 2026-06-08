import { open } from "@tauri-apps/plugin-dialog";
import { Button, Empty, Transfer } from "antd";
import type { TransferKey } from "antd/es/transfer/interface";
import type { TFunction } from "i18next";
import type { FC } from "react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import {
  addClipboardAppFromPath,
  deleteUnreferencedClipboardApps,
} from "@/commands";
import AssetImage from "@/components/AssetImage";
import Tooltip from "@/components/Tooltip";
import {
  mergeSourceApp,
  refreshSourceApps,
  removeSourceApps,
  sourceAppsState,
} from "@/stores/sourceApps";
import type { ClipboardApp } from "@/types/clipboard";
import type { Settings } from "@/types/settings";
import { isMac } from "@/utils/is";
import { log } from "@/utils/log";
import type {
  PreferenceSetting,
  PreferenceSettingChangeHandler,
} from "../types/preferences";
import PreferenceCountTag from "./PreferenceCountTag";

interface SourceAppsTransferProps {
  excludedAppsSetting: PreferenceSetting;
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
 * 来源应用复合设置：左侧展示可忽略应用，右侧维护已忽略应用。
 */
const SourceAppsTransfer: FC<SourceAppsTransferProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { excludedAppsSetting, settings, onChange } = props;
  const sourceApps = useSnapshot(sourceAppsState);
  const excludedAppIds = settings.clipboard.filters.excludedAppIds;
  const [adding, setAdding] = useState(false);

  const dataSource = useMemo(() => {
    return buildTransferItems(sourceApps.apps);
  }, [sourceApps.apps]);

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
   * 手动添加一个来源应用，成功后直接加入已忽略列表。
   */
  const handleAddApp = async () => {
    setAdding(true);

    try {
      const selected = await open({
        directory: false,
        filters: [
          {
            extensions: [isMac ? "app" : "exe"],
            name: t("schema.settings.source.appTransfer.appFileFilter"),
          },
        ],
        multiple: false,
        title: t("schema.settings.source.appTransfer.addApp"),
      });
      if (!selected) return;

      const added = await addClipboardAppFromPath(selected);
      mergeSourceApp(added);
      await onChange(
        excludedAppsSetting,
        mergeExcludedAppIds(excludedAppIds, added.id),
      );
    } catch (error) {
      log.warn("add source app failed", error);
    } finally {
      setAdding(false);
    }
  };

  /**
   * 手动触发可忽略应用刷新，完成后刷新穿梭框数据。
   */
  const handleRefreshApps = async () => {
    await refreshSourceApps(excludedAppIds);
  };

  /**
   * 根据穿梭框右侧 key 列表保存忽略应用。
   */
  const handleTransferChange = async (nextKeys: TransferKey[]) => {
    const nextExcludedAppIds = nextKeys.map((key) => {
      return String(key);
    });

    await onChange(excludedAppsSetting, nextExcludedAppIds);

    const removedIds = diffRemovedIds(excludedAppIds, nextExcludedAppIds);
    if (removedIds.length === 0) return;

    const deletedIds = await deleteUnreferencedClipboardApps(removedIds);
    removeSourceApps(deletedIds);
  };

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
        <Tooltip title={t("schema.settings.source.appTransfer.refreshApps")}>
          <Button
            aria-label={t("schema.settings.source.appTransfer.refreshApps")}
            icon={<i aria-hidden="true" className="i-lucide:refresh-cw" />}
            loading={sourceApps.loading}
            onClick={handleRefreshApps}
            size="small"
            type="text"
          />
        </Tooltip>
      </div>
    </div>
  );

  const targetTitle = (
    <span className="flex min-w-0 items-center justify-between gap-2">
      <span className="flex min-w-0 items-center gap-1">
        <span className="truncate">
          {t("schema.settings.source.appTransfer.ignoredTitle")}
        </span>
        <PreferenceCountTag>{targetCountLabel}</PreferenceCountTag>
      </span>
      <Tooltip title={t("schema.settings.source.appTransfer.addApp")}>
        <Button
          aria-label={t("schema.settings.source.appTransfer.addApp")}
          icon={<i aria-hidden="true" className="i-ph:plus-bold text-base" />}
          loading={adding}
          onClick={handleAddApp}
          size="small"
          type="text"
        />
      </Tooltip>
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
          header: "[&_.ant-transfer-list-header-selected]:hidden",
          root: "h-full",
          section: "h-full! flex-1",
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
        oneWay
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

/**
 * 把 Rust 返回的应用列表转换成 Transfer 数据。
 */
function buildTransferItems(
  apps: ReadonlyArray<ClipboardApp>,
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

  return Array.from(items.values()).sort((left, right) => {
    return left.title
      .toLocaleLowerCase()
      .localeCompare(right.title.toLocaleLowerCase());
  });
}

/**
 * 把手动添加的应用 id 追加到忽略列表，已存在时保持原列表不变。
 */
function mergeExcludedAppIds(excludedAppIds: string[], appId: string) {
  if (excludedAppIds.includes(appId)) return excludedAppIds;

  return [...excludedAppIds, appId];
}

/**
 * 计算这次从已忽略列表移除的应用 id。
 */
function diffRemovedIds(previousIds: string[], nextIds: string[]) {
  const next = new Set(nextIds);

  return previousIds.filter((id) => {
    return !next.has(id);
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
