import { relaunch } from "@tauri-apps/plugin-process";
import { useMount, useReactive } from "ahooks";
import {
  Button,
  Flex,
  Input,
  Modal,
  message,
  Select,
  Space,
  Table,
  Tooltip,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { filesize } from "filesize";
import { type Key, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import ProSwitch from "@/components/ProSwitch";
import UnoIcon from "@/components/UnoIcon";
import {
  cancelWebdavUpload,
  deleteWebdavBackup,
  getWebdavComputerName,
  getWebdavConfig,
  setWebdavConfig,
  testWebdavConfig,
} from "@/plugins/webdav";
import { clipboardStore } from "@/stores/clipboard";
import { formatDate } from "@/utils/dayjs";
import { wait } from "@/utils/shared";
import {
  backupToWebdav,
  getDefaultWebdavFilename,
  listWebdavBackupFiles,
  normalizeWebdavBackupFileName,
  restoreWebdavBackup,
} from "@/utils/webdavBackup";
import type { State } from "../..";

interface WebdavFormState {
  address: string;
  username: string;
  password: string;
  path: string;
}

interface BackupRow {
  fileName: string;
  size?: number;
  modified?: string;
}

const AUTO_BACKUP_OPTIONS = [
  { label: "关闭", value: 0 },
  { label: "1分钟", value: 1 },
  { label: "5分钟", value: 5 },
  { label: "15分钟", value: 15 },
  { label: "30分钟", value: 30 },
  { label: "1小时", value: 60 },
  { label: "2小时", value: 120 },
  { label: "6小时", value: 360 },
  { label: "12小时", value: 720 },
  { label: "24小时", value: 1440 },
];

const MAX_BACKUPS_OPTIONS = [
  { label: "无限制", value: 0 },
  { label: "1", value: 1 },
  { label: "3", value: 3 },
  { label: "5", value: 5 },
  { label: "10", value: 10 },
  { label: "20", value: 20 },
  { label: "50", value: 50 },
];

const Webdav = (props: { state: State }) => {
  const { state } = props;
  const { t } = useTranslation();
  const { webdav } = useSnapshot(clipboardStore);
  const [backupOpen, setBackupOpen] = useState(false);
  const [restoreOpen, setRestoreOpen] = useState(false);
  const [backupName, setBackupName] = useState("");
  const [computerName, setComputerName] = useState<string>();
  const [uploading, setUploading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [backups, setBackups] = useState<BackupRow[]>([]);
  const [selectedKeys, setSelectedKeys] = useState<Key[]>([]);
  const form = useReactive<WebdavFormState>({
    address: "",
    password: "",
    path: "",
    username: "",
  });

  useMount(async () => {
    const config = await getWebdavConfig();
    if (config) {
      form.address = config.address;
      form.username = config.username;
      form.password = config.password;
      form.path = config.path;
    }
    try {
      setBackupName(await getDefaultWebdavFilename());
    } catch (error: any) {
      message.error(String(error));
    }
  });

  const saveConfig = async () => {
    try {
      await setWebdavConfig({
        address: form.address,
        password: form.password,
        path: form.path,
        username: form.username,
      });
    } catch (error: any) {
      message.error(String(error));
    }
  };

  const handleTest = async () => {
    try {
      await saveConfig();
      await testWebdavConfig({
        address: form.address,
        password: form.password,
        path: form.path,
        username: form.username,
      });
      message.success(t("preference.data_backup.webdav.hints.test_success"));
    } catch (error: any) {
      message.error(
        t("preference.data_backup.webdav.hints.test_failed", {
          error: String(error),
        }),
      );
    }
  };

  const openBackupModal = async () => {
    setBackupOpen(true);
    try {
      const name = await getWebdavComputerName();
      setComputerName(name);
      setBackupName(await getDefaultWebdavFilename(name));
    } catch (error: any) {
      message.error(String(error));
      try {
        setBackupName(await getDefaultWebdavFilename(computerName));
      } catch (innerError: any) {
        message.error(String(innerError));
      }
    }
  };

  const updateStatus = (status: "success" | "error", error?: string) => {
    clipboardStore.webdav.lastBackupStatus = status;
    clipboardStore.webdav.lastBackupAt = formatDate();
    clipboardStore.webdav.lastBackupError = error;
  };

  const trimBackups = async () => {
    const maxBackups = clipboardStore.webdav.maxBackups;
    if (maxBackups <= 0) return;
    const list = await listWebdavBackupFiles();
    const sorted = list
      .map((item) => ({
        ...item,
        timeValue: parseTimeValue(item.fileName) || 0,
      }))
      .sort((a, b) => b.timeValue - a.timeValue);
    const excess = sorted.slice(maxBackups);
    if (excess.length === 0) return;
    await Promise.all(excess.map((item) => deleteWebdavBackup(item.fileName)));
  };

  const handleBackupConfirm = async () => {
    const name = backupName.trim();
    if (!name) return;
    const fileName = normalizeWebdavBackupFileName(name);

    try {
      setUploading(true);
      await saveConfig();
      await backupToWebdav(fileName, clipboardStore.webdav.slim);
      await trimBackups();
      updateStatus("success");
      message.success(t("preference.data_backup.webdav.hints.backup_success"));
      setBackupOpen(false);
    } catch (error: any) {
      updateStatus("error", String(error));
      message.error(String(error));
    } finally {
      setUploading(false);
    }
  };

  const handleBackupCancel = async () => {
    if (uploading) {
      await cancelWebdavUpload();
      setUploading(false);
    }
    setBackupOpen(false);
  };

  const loadBackups = async () => {
    try {
      setLoading(true);
      await saveConfig();
      const list = await listWebdavBackupFiles();
      const sorted = list
        .map((item) => ({
          ...item,
          timeValue:
            parseTimeValue(item.fileName) ||
            (item.modified ? Date.parse(item.modified) : 0),
        }))
        .sort((a, b) => b.timeValue - a.timeValue);
      setBackups(sorted);
      setSelectedKeys([]);
    } catch (error: any) {
      message.error(String(error));
    } finally {
      setLoading(false);
    }
  };

  const openRestoreModal = async () => {
    await loadBackups();
    setRestoreOpen(true);
  };

  const confirmRestore = async () => {
    return await new Promise<boolean>((resolve) => {
      Modal.confirm({
        cancelText: t("preference.data_backup.webdav.button.cancel_restore"),
        centered: true,
        content: t("preference.data_backup.webdav.hints.confirm_restore"),
        okText: t("preference.data_backup.webdav.button.confirm_restore"),
        onCancel: () => resolve(false),
        onOk: () => resolve(true),
      });
    });
  };

  const handleRestore = async (fileName: string) => {
    try {
      const confirmed = await confirmRestore();
      if (!confirmed) return;
      setRestoreOpen(false);
      state.spinning = true;
      await restoreWebdavBackup(fileName);
      message.success(t("preference.data_backup.webdav.hints.restore_success"));
      await wait(300);
      await relaunch();
    } catch (error: any) {
      message.error(String(error));
    } finally {
      state.spinning = false;
    }
  };

  const handleDelete = async (fileName: string) => {
    try {
      await deleteWebdavBackup(fileName);
      message.success(t("preference.data_backup.webdav.hints.delete_success"));
      await loadBackups();
    } catch (error: any) {
      message.error(String(error));
    }
  };

  const handleDeleteSelected = async () => {
    if (selectedKeys.length === 0) return;
    try {
      setLoading(true);
      await Promise.all(
        selectedKeys.map((name) => deleteWebdavBackup(String(name))),
      );
      message.success(t("preference.data_backup.webdav.hints.delete_success"));
      await loadBackups();
    } catch (error: any) {
      message.error(String(error));
    } finally {
      setLoading(false);
    }
  };

  const parseTimeValue = (fileName: string) => {
    const parts = fileName.split(".");
    const timestamp = parts[1];
    if (!timestamp || timestamp.length !== 14) return void 0;
    return Number(timestamp);
  };

  const parseTimeLabel = (fileName: string, fallback?: string) => {
    const parts = fileName.split(".");
    const timestamp = parts[1];
    if (!timestamp || timestamp.length !== 14) return fallback || "-";
    return `${timestamp.slice(0, 4)}-${timestamp.slice(4, 6)}-${timestamp.slice(6, 8)} ${timestamp.slice(8, 10)}:${timestamp.slice(10, 12)}:${timestamp.slice(12, 14)}`;
  };

  const parseDevice = (fileName: string) => {
    const parts = fileName.split(".");
    return parts[2] || "-";
  };

  const columns: ColumnsType<BackupRow> = [
    {
      dataIndex: "fileName",
      onHeaderCell: () => ({ className: "text-center" }),
      render: (value: string) => {
        return (
          <Space size={8}>
            <Button
              onClick={() => handleRestore(value)}
              size="small"
              type="link"
            >
              {t("preference.data_backup.webdav.button.restore")}
            </Button>
            <Button
              danger
              onClick={() => handleDelete(value)}
              size="small"
              type="link"
            >
              {t("preference.data_backup.webdav.button.delete")}
            </Button>
          </Space>
        );
      },
      title: t("preference.data_backup.webdav.table.action"),
    },
    {
      dataIndex: "fileName",
      ellipsis: true,
      onHeaderCell: () => ({ className: "text-center" }),
      render: (value: string) => value,
      title: t("preference.data_backup.webdav.table.name"),
    },
    {
      dataIndex: "fileName",
      onHeaderCell: () => ({ className: "text-center" }),
      render: (value: string, record) => parseTimeLabel(value, record.modified),
      title: t("preference.data_backup.webdav.table.time"),
    },
    {
      dataIndex: "size",
      onHeaderCell: () => ({ className: "text-center" }),
      render: (value?: number) => (value ? filesize(value) : "-"),
      title: t("preference.data_backup.webdav.table.size"),
    },
    {
      dataIndex: "fileName",
      onHeaderCell: () => ({ className: "text-center" }),
      render: (value: string) => parseDevice(value),
      title: t("preference.data_backup.webdav.table.device"),
    },
  ];

  const status = webdav.lastBackupStatus;
  const statusText =
    status === "success"
      ? t("preference.data_backup.webdav.status.success")
      : status === "error"
        ? t("preference.data_backup.webdav.status.failed")
        : t("preference.data_backup.webdav.status.never");
  const statusTime = webdav.lastBackupAt;
  const showStatusTime = status === "success" || status === "error";
  const statusError = webdav.lastBackupError;
  const statusIcon =
    status === "success"
      ? "i-lucide:check-circle-2"
      : status === "error"
        ? "i-lucide:triangle-alert"
        : "i-lucide:minus-circle";
  const statusColor =
    status === "success"
      ? "text-green-6"
      : status === "error"
        ? "text-red-6"
        : "text-color-3";

  const testLabel = t("preference.data_backup.webdav.button.test").replace(
    /\s+/g,
    "",
  );

  return (
    <>
      <ProList header={t("preference.data_backup.webdav.section_title")}>
        <ProListItem title={t("preference.data_backup.webdav.label.address")}>
          <Input
            onBlur={saveConfig}
            onChange={(event) => {
              form.address = event.target.value;
            }}
            onPressEnter={saveConfig}
            placeholder={t("preference.data_backup.webdav.placeholder.address")}
            style={{ width: 300 }}
            value={form.address}
          />
        </ProListItem>

        <ProListItem title={t("preference.data_backup.webdav.label.username")}>
          <Input
            onBlur={saveConfig}
            onChange={(event) => {
              form.username = event.target.value;
            }}
            onPressEnter={saveConfig}
            placeholder={t(
              "preference.data_backup.webdav.placeholder.username",
            )}
            style={{ width: 300 }}
            value={form.username}
          />
        </ProListItem>

        <ProListItem title={t("preference.data_backup.webdav.label.password")}>
          <div className="webdav-password">
            <Input.Password
              addonAfter={
                <div
                  className="webdav-test flex h-full cursor-pointer select-none items-center px-3"
                  onClick={handleTest}
                >
                  {testLabel}
                </div>
              }
              onBlur={saveConfig}
              onChange={(event) => {
                form.password = event.target.value;
              }}
              onPressEnter={saveConfig}
              placeholder={t(
                "preference.data_backup.webdav.placeholder.password",
              )}
              style={{ width: 300 }}
              value={form.password}
            />
          </div>
        </ProListItem>

        <ProListItem title={t("preference.data_backup.webdav.label.path")}>
          <Input
            onBlur={saveConfig}
            onChange={(event) => {
              form.path = event.target.value;
            }}
            onPressEnter={saveConfig}
            placeholder={t("preference.data_backup.webdav.placeholder.path")}
            style={{ width: 300 }}
            value={form.path}
          />
        </ProListItem>

        <ProSwitch
          checked={webdav.slim}
          description={t("preference.data_backup.webdav.hints.slim")}
          onChange={(value) => {
            clipboardStore.webdav.slim = value;
          }}
          title={t("preference.data_backup.webdav.label.slim")}
        />

        <ProListItem
          description={t("preference.data_backup.webdav.hints.manual")}
          title={t("preference.data_backup.webdav.label.manual")}
        >
          <Space>
            <Button onClick={openBackupModal}>
              💾 {t("preference.data_backup.webdav.button.backup")}
            </Button>
            <Button onClick={openRestoreModal}>
              📥 {t("preference.data_backup.webdav.button.restore_from")}
            </Button>
          </Space>
        </ProListItem>

        <ProListItem
          description={t("preference.data_backup.webdav.hints.auto_backup")}
          title={t("preference.data_backup.webdav.label.auto_backup")}
        >
          <Select
            onChange={(value) => {
              clipboardStore.webdav.autoBackup = value;
            }}
            options={AUTO_BACKUP_OPTIONS}
            style={{ width: 99 }}
            value={webdav.autoBackup}
          />
        </ProListItem>

        <ProListItem
          description={t("preference.data_backup.webdav.hints.max_backups")}
          title={t("preference.data_backup.webdav.label.max_backups")}
        >
          <Select
            onChange={(value) => {
              clipboardStore.webdav.maxBackups = value;
            }}
            options={MAX_BACKUPS_OPTIONS}
            style={{ width: 99 }}
            value={webdav.maxBackups}
          />
        </ProListItem>

        <ProListItem title={t("preference.data_backup.webdav.label.status")}>
          <Flex align="center" gap={8}>
            <UnoIcon className={statusColor} name={statusIcon} />
            <span className={statusColor}>
              {statusText}
              {showStatusTime && statusTime ? ` ${statusTime}` : ""}
            </span>
            {status === "error" && statusError && (
              <Tooltip
                title={t("preference.data_backup.webdav.hints.error", {
                  error: statusError,
                })}
              >
                <UnoIcon className="text-red-6" name="i-lucide:info" />
              </Tooltip>
            )}
          </Flex>
        </ProListItem>
      </ProList>

      <Modal
        closable
        maskClosable
        okButtonProps={{ loading: uploading }}
        okText={t("preference.data_backup.webdav.button.confirm_backup")}
        onCancel={handleBackupCancel}
        onOk={handleBackupConfirm}
        open={backupOpen}
        title={t("preference.data_backup.webdav.title.backup")}
      >
        <Space className="w-full" direction="vertical" size={8}>
          <span className="text-color-2 text-xs">
            {t("preference.data_backup.webdav.label.filename")}
          </span>
          <Input
            onChange={(event) => setBackupName(event.target.value)}
            placeholder={t(
              "preference.data_backup.webdav.placeholder.filename",
            )}
            value={backupName}
          />
        </Space>
      </Modal>

      <Modal
        closable
        footer={
          <Flex align="center" justify="space-between">
            <Flex align="center" gap={8}>
              <Button loading={loading} onClick={loadBackups}>
                {t("preference.data_backup.webdav.button.refresh")}
              </Button>
              <span className="text-color-2 text-xs">
                {t("preference.data_backup.webdav.label.total", {
                  total: backups.length,
                })}
              </span>
            </Flex>
            <Space>
              <Button
                danger
                disabled={selectedKeys.length === 0}
                onClick={handleDeleteSelected}
              >
                {t("preference.data_backup.webdav.button.delete_selected", {
                  total: selectedKeys.length,
                })}
              </Button>
              <Button onClick={() => setRestoreOpen(false)}>
                {t("preference.data_backup.webdav.button.close")}
              </Button>
            </Space>
          </Flex>
        }
        maskClosable
        onCancel={() => setRestoreOpen(false)}
        open={restoreOpen}
        title={t("preference.data_backup.webdav.title.restore")}
        width={860}
      >
        <Table
          columns={columns}
          dataSource={backups}
          loading={loading}
          pagination={{ pageSize: 10, size: "small" }}
          rowKey="fileName"
          rowSelection={{
            onChange: (keys) => setSelectedKeys(keys),
            selectedRowKeys: selectedKeys,
          }}
          scroll={{ y: 360 }}
          size="small"
          tableLayout="auto"
        />
      </Modal>
    </>
  );
};

export default Webdav;
