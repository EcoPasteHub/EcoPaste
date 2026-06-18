import { Alert, Checkbox, Form, Input, Modal, Radio } from "antd";
import type { FC } from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type BackupImportStrategy,
  type BackupReceivedPayload,
  type ImportHistoryBackupResult,
  importHistoryBackup,
} from "@/commands";
import { log } from "@/utils/log";

const OVERWRITE_READ_SECONDS = 3;

interface BackupImportModalProps {
  open: boolean;
  target: BackupReceivedPayload | null;
  onCancel: () => void;
  onImported: (result: ImportHistoryBackupResult) => void;
}

interface BackupImportForm {
  strategy: BackupImportStrategy;
  password?: string;
  overwriteConfirmed?: boolean;
}

/**
 * 采集备份导入策略和必要密码，并调用 Rust 执行导入。
 */
const BackupImportModal: FC<BackupImportModalProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { open, target, onCancel, onImported } = props;
  const [form] = Form.useForm<BackupImportForm>();
  const [loading, setLoading] = useState(false);
  const [overwriteReadSecondsLeft, setOverwriteReadSecondsLeft] = useState(
    OVERWRITE_READ_SECONDS,
  );
  const mode = target?.mode ?? "encrypted";
  const strategy = Form.useWatch("strategy", form) ?? "merge";
  const password = Form.useWatch("password", form);
  const overwriteConfirmed = Form.useWatch("overwriteConfirmed", form);
  const isEncryptedBackup = mode === "encrypted";
  const isOverwriteStrategy = strategy === "overwrite";
  const encryptedPasswordMissing = isEncryptedBackup && !password;
  const overwriteReadCompleted = overwriteReadSecondsLeft === 0;
  const overwriteSubmitDisabled =
    isOverwriteStrategy &&
    (!overwriteReadCompleted || overwriteConfirmed !== true);
  const submitDisabled =
    loading || !target || encryptedPasswordMissing || overwriteSubmitDisabled;
  const targetPath = target?.path ?? "";
  const backupName = targetPath.split(/[/\\]/).pop() ?? "";

  /**
   * 根据覆盖导入确认状态返回确认按钮文案。
   */
  const getOkText = () => {
    if (!isOverwriteStrategy) {
      return t("preferences:backup.import.ok");
    }
    if (!overwriteReadCompleted) {
      return t("preferences:backup.import.overwriteOkCountdown", {
        seconds: overwriteReadSecondsLeft,
      });
    }
    if (overwriteConfirmed !== true) {
      return t("preferences:backup.import.overwriteOkConfirm");
    }

    return t("preferences:backup.import.ok");
  };

  const okText = getOkText();

  useEffect(() => {
    if (!open || !targetPath) return;

    form.setFieldsValue({
      overwriteConfirmed: false,
      password: "",
      strategy: "merge",
    });
    setOverwriteReadSecondsLeft(OVERWRITE_READ_SECONDS);
  }, [form, open, targetPath]);

  useEffect(() => {
    if (!open || !isOverwriteStrategy) {
      return;
    }

    form.setFieldValue("overwriteConfirmed", false);
    setOverwriteReadSecondsLeft(OVERWRITE_READ_SECONDS);
  }, [form, isOverwriteStrategy, open]);

  useEffect(() => {
    if (!open || !isOverwriteStrategy || overwriteReadSecondsLeft === 0) {
      return;
    }

    const timer = window.setTimeout(() => {
      setOverwriteReadSecondsLeft((value) => {
        return Math.max(value - 1, 0);
      });
    }, 1000);

    return () => {
      window.clearTimeout(timer);
    };
  }, [isOverwriteStrategy, open, overwriteReadSecondsLeft]);

  /**
   * 取消导入并重置表单。
   */
  const resetAndCancel = () => {
    if (loading) return;

    form.resetFields();
    onCancel();
  };

  /**
   * 调用 Rust 导入备份，成功后交给父组件刷新偏好页状态。
   */
  const importBackup = async () => {
    if (!target) return;

    const values = await form.validateFields();

    setLoading(true);
    try {
      const result = await importHistoryBackup(
        {
          password: isEncryptedBackup ? values.password : void 0,
          path: target.path,
        },
        {
          strategy: values.strategy,
        },
      );

      form.resetFields();
      onImported(result);
    } catch (error) {
      log.warn("import backup failed", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      cancelText={t("common:actions.cancel")}
      confirmLoading={loading}
      destroyOnHidden
      okButtonProps={{
        danger: isOverwriteStrategy,
        disabled: submitDisabled,
      }}
      okText={okText}
      onCancel={resetAndCancel}
      onOk={importBackup}
      open={open}
      title={t("preferences:backup.import.title")}
    >
      <Form<BackupImportForm>
        form={form}
        initialValues={{
          overwriteConfirmed: false,
          password: "",
          strategy: "merge",
        }}
        layout="vertical"
      >
        <div className="mb-4 rounded-2 border border-ant-border-secondary bg-ant-fill-quaternary px-3 py-2.5">
          <div className="font-semibold text-sm">
            {backupName || t("preferences:backup.import.fileFallback")}
          </div>
          <div className="text-ant-secondary text-xs">
            {isEncryptedBackup
              ? t("preferences:backup.import.typeEncrypted")
              : t("preferences:backup.import.typePlain")}
          </div>
        </div>

        {isEncryptedBackup ? (
          <Form.Item
            label={t("preferences:backup.import.password")}
            name="password"
            rules={[
              {
                message: t("preferences:backup.import.passwordRequired"),
                required: true,
              },
            ]}
          >
            <Input.Password autoComplete="current-password" />
          </Form.Item>
        ) : null}

        <Form.Item
          label={t("preferences:backup.import.strategy")}
          name="strategy"
        >
          <Radio.Group>
            <Radio value="merge">
              {t("preferences:backup.import.strategyMerge")}
            </Radio>
            <Radio value="overwrite">
              {t("preferences:backup.import.strategyOverwrite")}
            </Radio>
          </Radio.Group>
        </Form.Item>

        {isOverwriteStrategy ? (
          <>
            <Alert
              className="mb-4"
              description={t("preferences:backup.import.overwriteWarning")}
              showIcon
              type="warning"
            />
            <Form.Item
              className="mt-3 mb-0"
              name="overwriteConfirmed"
              rules={[
                {
                  validator(_, value) {
                    if (value === true) {
                      return Promise.resolve();
                    }

                    return Promise.reject(
                      new Error(
                        t("preferences:backup.import.overwriteConfirmRequired"),
                      ),
                    );
                  },
                },
              ]}
              valuePropName="checked"
            >
              <Checkbox disabled={!overwriteReadCompleted}>
                {t("preferences:backup.import.overwriteConfirm")}
              </Checkbox>
            </Form.Item>
          </>
        ) : null}
      </Form>
    </Modal>
  );
};

export default BackupImportModal;
