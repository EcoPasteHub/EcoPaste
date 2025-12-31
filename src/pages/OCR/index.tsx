import { useMount, useReactive } from "ahooks";
import { Button, Divider, Flex, message, Spin, Tooltip } from "antd";
import clsx from "clsx";
import { writeText } from "tauri-plugin-clipboard-x-api";
import { useSnapshot } from "valtio";
import UnoIcon from "@/components/UnoIcon";
import { LISTEN_KEY } from "@/constants";
import { useTauriListen } from "@/hooks/useTauriListen";
import { hideWindow } from "@/plugins/window";
import { performOcr } from "@/services/ocr/openai";
import { translate } from "@/services/translate/openai";
import { globalStore } from "@/stores/global";
import styles from "./index.module.scss";

interface State {
  ocrText: string;
  translateText: string;
  ocrLoading: boolean;
  translateLoading: boolean;
  pinned: boolean;
  ocrError?: string;
  translateError?: string;
  sourceLanguage: string;
}

const INITIAL_STATE: State = {
  ocrError: undefined,
  ocrLoading: false,
  ocrText: "",
  pinned: false,
  sourceLanguage: "auto",
  translateError: undefined,
  translateLoading: false,
  translateText: "",
};

const OCR = () => {
  const { ocr } = useSnapshot(globalStore);
  const state = useReactive<State>(INITIAL_STATE);

  useMount(() => {
    state.pinned = ocr.windowPinned;
  });

  // 监听 OCR 开始事件
  useTauriListen<{ imageBase64: string; withTranslate: boolean }>(
    LISTEN_KEY.OCR_START,
    async ({ payload }) => {
      const { imageBase64, withTranslate } = payload;
      await handleOcr(imageBase64, withTranslate);
    },
  );

  // 执行 OCR
  const handleOcr = async (imageBase64: string, withTranslate: boolean) => {
    state.ocrLoading = true;
    state.ocrError = undefined;
    state.ocrText = "";
    state.translateText = "";
    state.translateError = undefined;

    try {
      const result = await performOcr(imageBase64, {
        apiBase: globalStore.ocr.apiBase,
        apiKey: globalStore.ocr.apiKey,
        model: globalStore.ocr.model,
        prompt: globalStore.ocr.prompt,
      });

      if (result.success) {
        state.ocrText = result.text;

        // 自动复制
        if (globalStore.ocr.autoCopy) {
          await writeText(result.text);
        }

        // 执行翻译
        if (withTranslate && globalStore.ocr.translate.enabled) {
          await handleTranslate(result.text);
        }
      } else {
        state.ocrError = result.error;
      }
    } catch (error) {
      state.ocrError = String(error);
    } finally {
      state.ocrLoading = false;
    }
  };

  // 执行翻译
  const handleTranslate = async (text: string) => {
    state.translateLoading = true;
    state.translateError = undefined;

    try {
      const translateConfig = globalStore.ocr.translate;
      const result = await translate(text, {
        apiBase: translateConfig.apiBase || globalStore.ocr.apiBase,
        apiKey: translateConfig.apiKey || globalStore.ocr.apiKey,
        model: translateConfig.model || globalStore.ocr.model,
        systemPrompt: translateConfig.systemPrompt,
        targetLanguage: translateConfig.targetLanguage,
      });

      if (result.success) {
        state.translateText = result.text;

        // 翻译完成后自动复制翻译结果
        if (globalStore.ocr.autoCopy) {
          await writeText(result.text);
        }
      } else {
        state.translateError = result.error;
      }
    } catch (error) {
      state.translateError = String(error);
    } finally {
      state.translateLoading = false;
    }
  };

  // 复制文本
  const handleCopy = async (text: string) => {
    await writeText(text);
    message.success("已复制");
  };

  // 切换固定状态
  const togglePinned = () => {
    state.pinned = !state.pinned;
    globalStore.ocr.windowPinned = state.pinned;
  };

  // 关闭窗口
  const handleClose = () => {
    hideWindow();
  };

  // 重新翻译
  const handleRetranslate = async () => {
    if (state.ocrText) {
      await handleTranslate(state.ocrText);
    }
  };

  return (
    <Flex className={styles.container} vertical>
      {/* 标题栏 */}
      <Flex
        align="center"
        className={styles.titleBar}
        data-tauri-drag-region
        justify="space-between"
      >
        <Tooltip title={state.pinned ? "取消固定" : "固定窗口"}>
          <Button
            className={clsx(styles.pinButton, {
              [styles.pinned]: state.pinned,
            })}
            icon={<UnoIcon name="i-lucide:pin" size={16} />}
            onClick={togglePinned}
            type="text"
          />
        </Tooltip>

        <Button
          className={styles.closeButton}
          icon={<UnoIcon name="i-lucide:x" size={16} />}
          onClick={handleClose}
          shape="circle"
          type="text"
        />
      </Flex>

      {/* OCR 结果区域 */}
      <Flex className={styles.resultSection} vertical>
        <Spin spinning={state.ocrLoading} tip="正在识别...">
          <div className={styles.textArea}>
            {state.ocrError ? (
              <div className={styles.error}>{state.ocrError}</div>
            ) : (
              <pre className={styles.text}>
                {state.ocrText || "等待截图..."}
              </pre>
            )}
          </div>
        </Spin>

        {/* OCR 操作按钮 */}
        <Flex align="center" className={styles.actionBar} gap="small">
          <Tooltip title="朗读">
            <Button
              disabled={!state.ocrText}
              icon={<UnoIcon name="i-lucide:volume-2" size={14} />}
              size="small"
              type="text"
            />
          </Tooltip>
          <Tooltip title="复制">
            <Button
              disabled={!state.ocrText}
              icon={<UnoIcon name="i-lucide:copy" size={14} />}
              onClick={() => handleCopy(state.ocrText)}
              size="small"
              type="text"
            />
          </Tooltip>
          <Tooltip title="重新识别">
            <Button
              disabled={!state.ocrText}
              icon={<UnoIcon name="i-lucide:refresh-cw" size={14} />}
              size="small"
              type="text"
            />
          </Tooltip>
          <Tooltip title="删除">
            <Button
              disabled={!state.ocrText}
              icon={<UnoIcon name="i-lucide:trash-2" size={14} />}
              onClick={() => {
                state.ocrText = "";
              }}
              size="small"
              type="text"
            />
          </Tooltip>

          <div className={styles.spacer} />

          <span className={styles.languageTag}>
            <UnoIcon name="i-lucide:circle" size={8} />
            {state.sourceLanguage === "auto"
              ? "自动检测"
              : state.sourceLanguage}
          </span>

          <Tooltip title="翻译">
            <Button
              disabled={!state.ocrText || state.translateLoading}
              icon={<UnoIcon name="i-lucide:languages" size={14} />}
              onClick={handleRetranslate}
              size="small"
              type="text"
            />
          </Tooltip>
        </Flex>
      </Flex>

      {/* 翻译结果区域（只有启用翻译时才显示） */}
      {ocr.translate.enabled && (
        <>
          <Divider className={styles.divider} />

          <Flex className={styles.resultSection} vertical>
            <Flex align="center" className={styles.modelInfo} gap="small">
              <UnoIcon name="i-lucide:bot" size={16} />
              <span>{ocr.translate.model || ocr.model}</span>

              <div className={styles.spacer} />

              <Button
                icon={<UnoIcon name="i-lucide:x" size={12} />}
                onClick={() => {
                  state.translateText = "";
                }}
                size="small"
                type="text"
              />
            </Flex>

            <Spin spinning={state.translateLoading} tip="正在翻译...">
              <div className={styles.textArea}>
                {state.translateError ? (
                  <div className={styles.error}>{state.translateError}</div>
                ) : (
                  <pre className={styles.text}>
                    {state.translateText || "等待翻译..."}
                  </pre>
                )}
              </div>
            </Spin>

            {/* 翻译操作按钮 */}
            <Flex align="center" className={styles.actionBar} gap="small">
              <Tooltip title="朗读">
                <Button
                  disabled={!state.translateText}
                  icon={<UnoIcon name="i-lucide:volume-2" size={14} />}
                  size="small"
                  type="text"
                />
              </Tooltip>
              <Tooltip title="复制">
                <Button
                  disabled={!state.translateText}
                  icon={<UnoIcon name="i-lucide:copy" size={14} />}
                  onClick={() => handleCopy(state.translateText)}
                  size="small"
                  type="text"
                />
              </Tooltip>
              <Tooltip title="重新翻译">
                <Button
                  disabled={!state.ocrText}
                  icon={<UnoIcon name="i-lucide:refresh-cw" size={14} />}
                  onClick={handleRetranslate}
                  size="small"
                  type="text"
                />
              </Tooltip>
            </Flex>
          </Flex>
        </>
      )}
    </Flex>
  );
};

export default OCR;
