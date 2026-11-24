import { useMount } from "ahooks";
import { Button, Popconfirm, Tag, Typography } from "antd";
import type { FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type SyncPeer,
  type SyncStatus,
  syncGetStatus,
  syncRemovePeer,
  syncRenamePeer,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import { cn } from "@/utils/cn";
import { log } from "@/utils/log";
import PairCodeInputModal from "./PairCodeInputModal";
import PairingCodeModal from "./PairingCodeModal";

interface DeviceRowProps {
  isSelf?: boolean;
  name: string;
  online?: boolean;
  onRemove?: () => void;
  onRename?: (name: string) => void;
  platform: SyncPeer["platform"];
}

/**
 * 设备列表中的一行：本机或已关联设备。
 */
const DeviceRow: FC<DeviceRowProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { isSelf, name, online, onRemove, onRename, platform } = props;

  return (
    <div className="flex items-center gap-3 border-ant-split border-t px-4 py-3 first:border-t-0">
      <i
        aria-hidden="true"
        className={cn("shrink-0 text-ant-secondary text-xl", {
          "i-lucide:laptop": !isSelf && platform === "macos",
          "i-lucide:monitor": isSelf || platform !== "macos",
        })}
      />

      <div className="flex min-w-0 flex-1 flex-col">
        {onRename ? (
          <Typography.Text
            className="text-ant-text text-sm"
            editable={{ onChange: onRename, triggerType: ["icon", "text"] }}
          >
            {name}
          </Typography.Text>
        ) : (
          <span className="truncate text-ant-text text-sm">{name}</span>
        )}
        <span className="text-ant-tertiary text-xs">
          {isSelf ? t("sync.devices.self") : t(`sync.devices.${platform}`)}
        </span>
      </div>

      {!isSelf && (
        <Tag color={online ? "success" : "default"} variant="filled">
          {online ? t("sync.devices.online") : t("sync.devices.offline")}
        </Tag>
      )}

      {onRemove && (
        <Popconfirm
          cancelText={t("sync.devices.cancel")}
          okText={t("sync.devices.unlink")}
          onConfirm={onRemove}
          title={t("sync.devices.unlinkConfirm", { name })}
        >
          <Button danger size="small" type="text">
            {t("sync.devices.unlink")}
          </Button>
        </Popconfirm>
      )}
    </div>
  );
};

/**
 * 「我的设备」分组：展示本机与已关联设备，提供匹配码入口。
 */
const SyncDevicesPanel: FC = () => {
  const { t } = useTranslation("preferences");
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [codeOpen, setCodeOpen] = useState(false);
  const [inputOpen, setInputOpen] = useState(false);

  const running = status?.running === true;

  const refresh = async () => {
    try {
      setStatus(await syncGetStatus());
    } catch (error) {
      log.warn("load sync status failed", error);
    }
  };

  const handleShowCode = () => {
    setCodeOpen(true);
  };

  const handleLink = () => {
    setInputOpen(true);
  };

  const handleCloseCode = () => {
    setCodeOpen(false);
  };

  const handleCloseInput = () => {
    setInputOpen(false);
  };

  const handleRename = async (id: string, name: string) => {
    try {
      await syncRenamePeer(id, name);
    } catch (error) {
      log.warn("rename sync device failed", error);
    }
  };

  const handleRemove = async (id: string) => {
    try {
      await syncRemovePeer(id);
    } catch (error) {
      log.warn("remove sync device failed", error);
    }
  };

  useMount(async () => {
    await refresh();
  });

  useTauriListen(TAURI_EVENT.SYNC_DEVICES, () => {
    void refresh();
  });

  useTauriListen(TAURI_EVENT.SETTINGS_UPDATED, () => {
    void refresh();
  });

  return (
    <div>
      <div className="flex items-center justify-between gap-3 px-4 py-3">
        <span className="text-ant-secondary text-xs">
          {running ? t("sync.devices.hint") : t("sync.devices.disabled")}
        </span>

        <div className="flex shrink-0 gap-2">
          <Button disabled={!running} onClick={handleLink} size="small">
            {t("sync.devices.link")}
          </Button>
          <Button
            disabled={!running}
            onClick={handleShowCode}
            size="small"
            type="primary"
          >
            {t("sync.devices.showCode")}
          </Button>
        </div>
      </div>

      {status && (
        <div className="border-ant-split border-t">
          <DeviceRow
            isSelf
            name={status.deviceName}
            platform={status.platform}
          />
          {status.peers.map((peer) => {
            const handlePeerRemove = () => {
              void handleRemove(peer.id);
            };
            const handlePeerRename = (name: string) => {
              void handleRename(peer.id, name);
            };

            return (
              <DeviceRow
                key={peer.id}
                name={peer.name}
                online={peer.online}
                onRemove={handlePeerRemove}
                onRename={handlePeerRename}
                platform={peer.platform}
              />
            );
          })}
        </div>
      )}

      <PairingCodeModal onClose={handleCloseCode} open={codeOpen} />
      <PairCodeInputModal onClose={handleCloseInput} open={inputOpen} />
    </div>
  );
};

export default SyncDevicesPanel;
