import { emit } from "@tauri-apps/api/event";
import { downloadDir } from "@tauri-apps/api/path";
import { confirm, open } from "@tauri-apps/plugin-dialog";
import { revealItemInDir } from "@tauri-apps/plugin-opener";
import { Flex, List, message } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { compress, decompress, fullName } from "tauri-plugin-fs-pro-api";
import ProList from "@/components/ProList";
import UnoIcon from "@/components/UnoIcon";
import { LISTEN_KEY } from "@/constants";
import { showWindow } from "@/plugins/window";
import { globalStore } from "@/stores/global";
import { dayjs, formatDate } from "@/utils/dayjs";
import {
  getSaveDatabasePath,
  getSaveDataPath,
  getSaveImagePath,
  getSaveStorePath,
  join,
} from "@/utils/path";
import { wait } from "@/utils/shared";
import { restoreStore, saveStore } from "@/utils/store";
import type { State } from "../..";

const Manual: FC<{ state: State }> = (props) => {
  const { state } = props;
  const { t } = useTranslation();

  // 备份文件的扩展名
  const extname = () => {
    return `${globalStore.env.appName}-backup`;
  };

  // 导入数据
  const handleImport = async () => {
    try {
      const confirmed = await confirm(
        t("preference.data_backup.import_export.hints.confirm_import"),
        {
          cancelLabel: t("preference.data_backup.import_export.button.cancel"),
          kind: "warning",
          okLabel: t(
            "preference.data_backup.import_export.button.confirm_import",
          ),
          title: t("preference.data_backup.import_export.label.confirm_import"),
        },
      );

      if (!confirmed) return;

      const path = await open({
        filters: [{ extensions: [extname()], name: "" }],
      });

      showWindow();

      if (!path) return;

      state.spinning = true;

      emit(LISTEN_KEY.CLOSE_DATABASE);

      await wait();

      await decompress(path, getSaveDataPath());

      await restoreStore(true);

      emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);

      message.success(
        t("preference.data_backup.import_export.hints.import_success"),
      );
    } catch (error) {
      message.error(String(error));
    } finally {
      state.spinning = false;
    }
  };

  // 导出数据
  const handleExport = async () => {
    try {
      const confirmed = await confirm(
        t("preference.data_backup.import_export.hints.confirm_export"),
        {
          cancelLabel: t("preference.data_backup.import_export.button.cancel"),
          kind: "warning",
          okLabel: t(
            "preference.data_backup.import_export.button.confirm_export",
          ),
          title: t("preference.data_backup.import_export.label.confirm_export"),
        },
      );

      if (!confirmed) return;

      state.spinning = true;

      await saveStore(true);

      const filename = formatDate(dayjs(), "YYYY_MM_DD_HH_mm_ss");

      const path = join(await downloadDir(), `${filename}.${extname()}`);

      await compress(getSaveDataPath(), path, {
        includes: [
          await fullName(getSaveImagePath()),
          await fullName(await getSaveDatabasePath()),
          await fullName(await getSaveStorePath(true)),
        ],
      });

      await revealItemInDir(path);

      message.success(
        t("preference.data_backup.import_export.hints.export_success"),
      );
    } catch (error) {
      message.error(String(error));
    } finally {
      state.spinning = false;
    }
  };

  const mapList = [
    {
      event: handleImport,
      icon: "i-hugeicons:database-import",
      label: t("preference.data_backup.import_export.button.import"),
    },
    {
      event: handleExport,
      icon: "i-hugeicons:database-export",
      label: t("preference.data_backup.import_export.button.export"),
    },
  ];

  return (
    <ProList header={t("preference.data_backup.import_export.title")}>
      <List.Item className="p-4!">
        <Flex className="w-full" gap="middle">
          {mapList.map((item) => {
            const { label, icon, event } = item;

            return (
              <Flex
                align="center"
                className="b b-color-2 hover:b-primary h-25.5 flex-1 cursor-pointer rounded-lg bg-color-3 px-2 text-center transition hover:text-primary"
                gap="small"
                justify="center"
                key={label}
                onClick={event}
                vertical
              >
                <UnoIcon name={icon} size={26} />
                {label}
              </Flex>
            );
          })}
        </Flex>
      </List.Item>
    </ProList>
  );
};

export default Manual;
