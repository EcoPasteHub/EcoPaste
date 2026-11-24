import { Button, Modal } from "antd";
import type { FC } from "react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type SyncPairingCode,
  type SyncPairingEvent,
  syncCancelPairing,
  syncGetStatus,
  syncStartPairing,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import { getMessageApi } from "@/utils/feedback";
import { log } from "@/utils/log";

interface PairingCodeModalProps {
  open: boolean;
  onClose: () => void;
}

/**
 * 展示本机 6 位匹配码，倒计时到期后可重新生成；有设备输入成功后自动关闭。
 */
const PairingCodeModal: FC<PairingCodeModalProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { open, onClose } = props;
  const [pairing, setPairing] = useState<SyncPairingCode | null>(null);
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [addresses, setAddresses] = useState<string[]>([]);
  const openRef = useRef(open);
  openRef.current = open;

  const digits = (pairing?.code ?? "").padEnd(6, " ").split("");
  const expired = pairing !== null && secondsLeft <= 0;

  const generate = async () => {
    try {
      setPairing(await syncStartPairing());
    } catch (error) {
      log.warn("start sync pairing failed", error);
      onClose();
      return;
    }

    try {
      const status = await syncGetStatus();

      setAddresses(status.addresses);
    } catch (error) {
      log.warn("load local sync addresses failed", error);
    }
  };

  const handleRegenerate = async () => {
    await generate();
  };

  const handleCancel = async () => {
    try {
      await syncCancelPairing();
    } catch (error) {
      log.warn("cancel sync pairing failed", error);
    }
    onClose();
  };

  useTauriListen<SyncPairingEvent>(TAURI_EVENT.SYNC_PAIRING, (event) => {
    if (!openRef.current) return;

    if (event.payload.state === "success") {
      getMessageApi().success(
        t("sync.pairing.success", { name: event.payload.peer ?? "" }),
      );
      setPairing(null);
      onClose();
    }
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: 仅在弹窗开关时生成 / 清理匹配码
  useEffect(() => {
    if (!open) {
      setPairing(null);
      return;
    }

    void generate();
  }, [open]);

  useEffect(() => {
    if (!pairing) return;

    const tick = () => {
      setSecondsLeft(
        Math.max(0, Math.ceil((pairing.expiresAt - Date.now()) / 1000)),
      );
    };
    tick();

    const timer = window.setInterval(tick, 1000);
    return () => {
      window.clearInterval(timer);
    };
  }, [pairing]);

  const minutes = Math.floor(secondsLeft / 60);
  const seconds = String(secondsLeft % 60).padStart(2, "0");

  return (
    <Modal
      centered
      footer={null}
      onCancel={handleCancel}
      open={open}
      title={t("sync.pairing.title")}
      width={420}
    >
      <div className="flex flex-col items-center gap-4 py-2">
        <div className="text-ant-secondary text-sm">
          {expired
            ? t("sync.pairing.expired")
            : t("sync.pairing.validFor", { time: `${minutes}:${seconds}` })}
        </div>

        <div className="flex gap-2">
          {digits.map((digit, index) => {
            return (
              <div
                className="flex h-14 w-10 items-center justify-center rounded-2 border border-ant-border bg-ant-layout font-semibold text-2xl text-ant-primary tabular-nums"
                // biome-ignore lint/suspicious/noArrayIndexKey: 固定 6 位槽位，顺序即身份
                key={index}
              >
                {expired ? "" : digit.trim()}
              </div>
            );
          })}
        </div>

        <div className="text-center text-ant-secondary text-xs leading-relaxed">
          {t("sync.pairing.hint")}
        </div>

        {addresses.length > 0 && (
          <div className="text-center text-ant-secondary text-xs">
            {t("sync.pairing.addresses", { addresses: addresses.join(" / ") })}
          </div>
        )}

        {expired && (
          <Button onClick={handleRegenerate} type="primary">
            {t("sync.pairing.regenerate")}
          </Button>
        )}
      </div>
    </Modal>
  );
};

export default PairingCodeModal;
