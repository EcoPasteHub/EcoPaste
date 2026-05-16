import { relaunch } from "@tauri-apps/plugin-process";
import { check, type Update } from "@tauri-apps/plugin-updater";
import { useCreation, useReactive } from "ahooks";
import { Flex, Modal, message, Typography } from "antd";
import clsx from "clsx";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import Markdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import { GITHUB_LINK, LISTEN_KEY, UPDATE_MESSAGE_KEY } from "@/constants";
import { useImmediateKey } from "@/hooks/useImmediateKey";
import { useTauriListen } from "@/hooks/useTauriListen";
import { showWindow } from "@/plugins/window";
import { globalStore } from "@/stores/global";
import type { Interval } from "@/types/shared";
import { dayjs, formatDate } from "@/utils/dayjs";
import styles from "./index.module.scss";

const { Link, Text } = Typography;

interface State {
  open?: boolean;
  loading?: boolean;
  update?: Update;
  total?: number;
  download: number;
}

const UpdateApp = () => {
  const { t } = useTranslation();
  const timerRef = useRef<Interval>();
  const state = useReactive<State>({ download: 0 });
  const [messageApi, contextHolder] = message.useMessage();

  // 监听自动更新配置变化
  useImmediateKey(globalStore.update, "auto", (value) => {
    clearInterval(timerRef.current);

    if (!value) return;

    checkUpdate();

    timerRef.current = setInterval(checkUpdate, 1000 * 60 * 60 * 24);
  });

  // 监听参与测试版本配置变化
  useImmediateKey(globalStore.update, "beta", (value) => {
    if (!value) return;

    checkUpdate();
  });

  // 监听更新事件
  useTauriListen<boolean>(LISTEN_KEY.UPDATE_APP, () => {
    checkUpdate(true);

    messageApi.open({
      content: t("component.app_update.hints.checking_update"),
      duration: 0,
      key: UPDATE_MESSAGE_KEY,
      type: "loading",
    });
  });

  // 确认按钮的文字
  const okText = useCreation(() => {
    const { loading, total, download } = state;

    if (loading) {
      if (!total) return "0%";

      const percent = (download / total) * 100;

      return `${percent.toFixed(2)}%`;
    }

    return t("component.app_update.button.confirm_update");
  }, [state.loading, state.total, state.download]);

  // 检查更新
  const checkUpdate = async (showMessage = false) => {
    try {
      const update = await check({
        headers: {
          "join-beta": String(globalStore.update.beta),
        },
      });

      if (update) {
        showWindow();

        const { version, currentVersion, body = "", date } = update;

        Object.assign(update, {
          body: replaceBody(body),
          currentVersion: `v${currentVersion}`,
          date: formatDate(dayjs.utc(date?.split(".")[0]).local()),
          version: `v${version}`,
        });

        Object.assign(state, { open: true, update });

        messageApi.destroy(UPDATE_MESSAGE_KEY);
      } else if (showMessage) {
        messageApi.open({
          content: t("component.app_update.hints.latest_version"),
          key: UPDATE_MESSAGE_KEY,
          type: "success",
        });
      }
    } catch (error) {
      if (!showMessage) return;

      messageApi.open({
        content: String(error),
        key: UPDATE_MESSAGE_KEY,
        type: "error",
      });
    }
  };

  // 替换更新日志里的内容
  const replaceBody = (body: string) => {
    return body
      .replace(/&nbsp;/g, "")
      .split("\n")
      .map((line) => line.replace(/\s*-\s+by\s+@.*/, ""))
      .join("\n");
  };

  const handleOk = async () => {
    state.loading = true;

    await state.update?.downloadAndInstall((progress) => {
      switch (progress.event) {
        case "Started":
          state.total = progress.data.contentLength;
          break;
        case "Progress":
          state.download += progress.data.chunkLength;
          break;
      }
    });

    state.loading = false;

    relaunch();
  };

  const handleCancel = () => {
    state.open = false;
  };

  return (
    <>
      {contextHolder}

      <Modal
        cancelButtonProps={{ disabled: state.loading }}
        cancelText={t("component.app_update.button.cancel_update")}
        centered
        className={styles.modal}
        closable={false}
        confirmLoading={state.loading}
        destroyOnClose
        keyboard={false}
        maskClosable={false}
        okText={okText}
        onCancel={handleCancel}
        onOk={handleOk}
        open={state.open}
        title={t("component.app_update.label.new_version_title")}
      >
        <Flex className="pt-1" gap="small" vertical>
          <Flex align="center">
            {t("component.app_update.label.release_version")}：
            <span>
              {state.update?.currentVersion} 👉{" "}
              <Link
                href={`${GITHUB_LINK}/releases/tag/${state.update?.version}`}
              >
                {state.update?.version}
              </Link>
            </span>
          </Flex>

          <Flex align="center">
            {t("component.app_update.label.release_time")}：
            <span>{state.update?.date}</span>
          </Flex>

          <Flex vertical>
            {t("component.app_update.label.release_notes")}：
            <Markdown
              className={clsx(styles.markdown, "max-h-50 overflow-auto")}
              components={{
                a: ({ href, children }) => <Link href={href}>{children}</Link>,
                code: ({ children }) => <Text code>{children}</Text>,
                mark: ({ children }) => <Text mark>{children}</Text>,
              }}
              rehypePlugins={[rehypeRaw]}
            >
              {state.update?.body}
            </Markdown>
          </Flex>
        </Flex>
      </Modal>
    </>
  );
};

export default UpdateApp;
