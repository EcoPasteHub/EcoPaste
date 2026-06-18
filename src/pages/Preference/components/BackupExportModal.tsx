import { save } from "@tauri-apps/plugin-dialog";
import { Alert, Checkbox, Form, Input, Modal, Segmented } from "antd";
import type { FC } from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type BackupExportMode,
  type ExportHistoryBackupResult,
  exportHistoryBackup,
} from "@/commands";
import { log } from "@/utils/log";

const BACKUP_EXTENSION = "ecopastebak";
const PLAIN_READ_SECONDS = 3;

interface BackupExportModalProps {
  open: boolean;
  onCancel: () => void;
  onExported: (result: ExportHistoryBackupResult) => void;
}

interface BackupExportForm {
  mode: BackupExportMode;
  password?: string;
  confirmPassword?: string;
  plainConfirmed?: boolean;
}

/**
 * 采集备份导出模式和密码，并把保存路径交给 Rust 生成 `.ecopastebak`。
 */
const BackupExportModal: FC<BackupExportModalProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { open, onCancel, onExported } = props;
  const [form] = Form.useForm<BackupExportForm>();
  const [loading, setLoading] = useState(false);
  const [plainReadSecondsLeft, setPlainReadSecondsLeft] =
    useState(PLAIN_READ_SECONDS);
  const mode = Form.useWatch("mode", form) ?? "encrypted";
  const password = Form.useWatch("password", form);
  const confirmPassword = Form.useWatch("confirmPassword", form);
  const plainConfirmed = Form.useWatch("plainConfirmed", form);
  const isEncryptedMode = mode === "encrypted";
  const isPlainMode = mode === "plain";
  const plainReadCompleted = plainReadSecondsLeft === 0;
  const encryptedSubmitDisabled =
    isEncryptedMode &&
    (!password ||
      password.length < 8 ||
      !confirmPassword ||
      password !== confirmPassword);
  const plainSubmitDisabled =
    isPlainMode && (!plainReadCompleted || plainConfirmed !== true);
  const submitDisabled =
    loading || encryptedSubmitDisabled || plainSubmitDisabled;

  /**
   * 根据明文备份阅读确认状态返回确认按钮文案。
   */
  const getOkText = () => {
    if (!isPlainMode) {
      return t("preferences:backup.export.ok");
    }
    if (!plainReadCompleted) {
      return t("preferences:backup.export.okReadCountdown", {
        seconds: plainReadSecondsLeft,
      });
    }
    if (plainConfirmed !== true) {
      return t("preferences:backup.export.okReadConfirm");
    }

    return t("preferences:backup.export.ok");
  };

  const okText = getOkText();

  useEffect(() => {
    if (!open || !isPlainMode) {
      return;
    }

    setPlainReadSecondsLeft(PLAIN_READ_SECONDS);
    form.setFieldValue("plainConfirmed", false);
  }, [form, isPlainMode, open]);

  useEffect(() => {
    if (!open || !isPlainMode || plainReadSecondsLeft === 0) {
      return;
    }

    const timer = window.setTimeout(() => {
      setPlainReadSecondsLeft((value) => {
        return Math.max(value - 1, 0);
      });
    }, 1000);

    return () => {
      window.clearTimeout(timer);
    };
  }, [isPlainMode, open, plainReadSecondsLeft]);

  /**
   * 取消导出并重置表单。
   */
  const resetAndCancel = () => {
    if (loading) return;

    form.resetFields();
    onCancel();
  };

  /**
   * 打开保存对话框并生成带秒级时间戳的默认备份名。
   */
  const pickTargetPath = async () => {
    const now = new Date();
    const stamp = [
      now.getFullYear(),
      String(now.getMonth() + 1).padStart(2, "0"),
      String(now.getDate()).padStart(2, "0"),
      "-",
      String(now.getHours()).padStart(2, "0"),
      String(now.getMinutes()).padStart(2, "0"),
      String(now.getSeconds()).padStart(2, "0"),
    ].join("");

    return await save({
      defaultPath: `EcoPaste-Backup-${stamp}.${BACKUP_EXTENSION}`,
      filters: [
        {
          extensions: [BACKUP_EXTENSION],
          name: t("preferences:backup.fileFilter"),
        },
      ],
    });
  };

  /**
   * 校验导出表单，选择目标路径后调用 Rust 写入备份包。
   */
  const exportBackup = async () => {
    const values = await form.validateFields();
    const targetPath = await pickTargetPath();
    if (!targetPath) return;

    setLoading(true);
    try {
      const result = await exportHistoryBackup(targetPath, {
        mode: values.mode,
        password: isEncryptedMode ? values.password : void 0,
      });

      form.resetFields();
      onExported(result);
    } catch (error) {
      log.warn("export backup failed", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      cancelText={t("common:actions.cancel")}
      confirmLoading={loading}
      destroyOnHidden
      okButtonProps={{ disabled: submitDisabled }}
      okText={okText}
      onCancel={resetAndCancel}
      onOk={exportBackup}
      open={open}
      title={t("preferences:backup.export.title")}
    >
      <Form<BackupExportForm>
        form={form}
        initialValues={{ mode: "encrypted" }}
        layout="vertical"
      >
        <Form.Item name="mode">
          <Segmented
            block
            options={[
              {
                label: t("preferences:backup.export.modeEncrypted"),
                value: "encrypted",
              },
              {
                label: t("preferences:backup.export.modePlain"),
                value: "plain",
              },
            ]}
          />
        </Form.Item>

        {isEncryptedMode ? (
          <>
            <Alert
              className="mb-4"
              showIcon
              title={t("preferences:backup.export.encryptedHint")}
              type="info"
            />
            <Form.Item
              label={t("preferences:backup.export.password")}
              name="password"
              rules={[
                {
                  message: t("preferences:backup.export.passwordMin"),
                  min: 8,
                },
                {
                  message: t("preferences:backup.export.passwordRequired"),
                  required: true,
                },
              ]}
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>
            <Form.Item
              dependencies={["password"]}
              label={t("preferences:backup.export.confirmPassword")}
              name="confirmPassword"
              rules={[
                {
                  message: t("preferences:backup.export.confirmRequired"),
                  required: true,
                },
                ({ getFieldValue }) => {
                  return {
                    validator(_, value) {
                      if (!value || getFieldValue("password") === value) {
                        return Promise.resolve();
                      }

                      return Promise.reject(
                        new Error(
                          t("preferences:backup.export.passwordMismatch"),
                        ),
                      );
                    },
                  };
                },
              ]}
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>
          </>
        ) : (
          <>
            <Alert
              className="mb-4"
              description={t("preferences:backup.export.plainWarning")}
              showIcon
              title={t("preferences:backup.export.plainWarningTitle")}
              type="warning"
            />
            <Form.Item
              className="mt-3 mb-0"
              name="plainConfirmed"
              rules={[
                {
                  validator(_, value) {
                    if (value === true) {
                      return Promise.resolve();
                    }

                    return Promise.reject(
                      new Error(
                        t("preferences:backup.export.plainConfirmRequired"),
                      ),
                    );
                  },
                },
              ]}
              valuePropName="checked"
            >
              <Checkbox disabled={!plainReadCompleted}>
                {t("preferences:backup.export.plainConfirm")}
              </Checkbox>
            </Form.Item>
          </>
        )}
      </Form>
    </Modal>
  );
};

export default BackupExportModal;
