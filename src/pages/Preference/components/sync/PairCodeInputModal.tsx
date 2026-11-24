import { Button, Input, Modal } from "antd";
import type { ChangeEvent, FC } from "react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { syncPairWithCode } from "@/commands";
import { cn } from "@/utils/cn";
import { getMessageApi } from "@/utils/feedback";
import { log } from "@/utils/log";

const CODE_LENGTH = 6;

interface PairCodeInputModalProps {
  open: boolean;
  onClose: () => void;
}

/**
 * 输入另一台设备上展示的 6 位匹配码，输满即自动提交。
 * 局域网发现被网络拦截时，可展开填写对方 IP 直接连接。
 */
const PairCodeInputModal: FC<PairCodeInputModalProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { open, onClose } = props;
  const [code, setCode] = useState("");
  const [host, setHost] = useState("");
  const [manual, setManual] = useState(false);
  const [loading, setLoading] = useState(false);
  const [focused, setFocused] = useState(false);
  const codeInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (!open || manual) return;

    const timer = window.setTimeout(() => {
      codeInputRef.current?.focus();
    }, 120);

    return () => {
      window.clearTimeout(timer);
    };
  }, [open, manual]);

  useEffect(() => {
    if (!open) {
      setCode("");
      setHost("");
      setManual(false);
      setLoading(false);
    }
  }, [open]);

  const submit = async (value: string) => {
    setLoading(true);

    try {
      const peer = await syncPairWithCode(
        value,
        manual && host.trim().length > 0 ? host.trim() : void 0,
      );

      getMessageApi().success(t("sync.pairing.success", { name: peer.name }));
      onClose();
    } catch (error) {
      log.warn("pair with code failed", error);
      setCode("");
    } finally {
      setLoading(false);
    }
  };

  const handleCodeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value.replace(/\D/g, "").slice(0, CODE_LENGTH);

    setCode(value);

    if (value.length === CODE_LENGTH) {
      void submit(value);
    }
  };

  const handleHostChange = (event: ChangeEvent<HTMLInputElement>) => {
    setHost(event.target.value);
  };

  const handleHostEnter = () => {
    if (code.length !== CODE_LENGTH) return;

    void submit(code);
  };

  const showManual = () => {
    setManual(true);
  };

  const handleCancel = () => {
    if (loading) return;

    onClose();
  };

  const handleCodeFocus = () => {
    setFocused(true);
  };

  const handleCodeBlur = () => {
    setFocused(false);
  };

  const cells = Array.from({ length: CODE_LENGTH }, (_, index) => {
    return code[index] ?? "";
  });

  return (
    <Modal
      centered
      footer={null}
      onCancel={handleCancel}
      open={open}
      title={t("sync.input.title")}
      width={420}
    >
      <div className="flex flex-col items-center gap-4 py-2">
        <div className="text-center text-ant-secondary text-sm leading-relaxed">
          {t("sync.input.hint")}
        </div>

        {manual && (
          <div className="flex w-full flex-col gap-1.5">
            <span className="text-ant-secondary text-xs">
              {t("sync.input.hostLabel")}
            </span>
            <Input
              allowClear
              autoFocus
              disabled={loading}
              onChange={handleHostChange}
              onPressEnter={handleHostEnter}
              placeholder={t("sync.input.hostPlaceholder")}
              value={host}
            />
          </div>
        )}

        <div className="relative flex gap-2">
          {cells.map((digit, index) => {
            const active = focused && index === Math.min(code.length, 5);

            return (
              <div
                className={cn(
                  "flex h-12 w-10 items-center justify-center rounded-2 border bg-ant-layout font-semibold text-ant-text text-xl tabular-nums",
                  active ? "border-ant-primary" : "border-ant-border",
                )}
                // biome-ignore lint/suspicious/noArrayIndexKey: 固定 6 位槽位，顺序即身份
                key={index}
              >
                {digit}
              </div>
            );
          })}

          <input
            aria-label={t("sync.input.title")}
            autoComplete="off"
            className="absolute inset-0 size-full cursor-text opacity-0"
            disabled={loading}
            inputMode="numeric"
            maxLength={CODE_LENGTH}
            onBlur={handleCodeBlur}
            onChange={handleCodeChange}
            onFocus={handleCodeFocus}
            ref={codeInputRef}
            value={code}
          />
        </div>

        {manual ? (
          <span className="text-center text-ant-secondary text-xs leading-relaxed">
            {t("sync.input.hostHint")}
          </span>
        ) : (
          <Button onClick={showManual} size="small" type="link">
            {t("sync.input.manual")}
          </Button>
        )}
      </div>
    </Modal>
  );
};

export default PairCodeInputModal;
