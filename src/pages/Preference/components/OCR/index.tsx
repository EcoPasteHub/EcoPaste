import { Input, Select } from "antd";
import TextArea from "antd/es/input/TextArea";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import ProShortcut from "@/components/ProShortcut";
import ProSwitch from "@/components/ProSwitch";
import { globalStore } from "@/stores/global";

const LANGUAGE_OPTIONS = [
  { label: "简体中文", value: "zh-CN" },
  { label: "繁體中文", value: "zh-TW" },
  { label: "English", value: "en" },
  { label: "日本語", value: "ja" },
  { label: "한국어", value: "ko" },
  { label: "Français", value: "fr" },
  { label: "Deutsch", value: "de" },
  { label: "Español", value: "es" },
  { label: "Русский", value: "ru" },
  { label: "Português", value: "pt" },
  { label: "Italiano", value: "it" },
  { label: "العربية", value: "ar" },
];

const OCRSettings = () => {
  const { ocr, shortcut } = useSnapshot(globalStore);
  const { t } = useTranslation();

  return (
    <>
      {/* API 设置 */}
      <ProList header={t("preference.ocr.api_settings.title")}>
        <ProListItem
          description={t("preference.ocr.api_settings.hints.api_base")}
          title={t("preference.ocr.api_settings.label.api_base")}
        >
          <Input
            onChange={(e) => {
              globalStore.ocr.apiBase = e.target.value;
            }}
            placeholder="https://api.openai.com"
            style={{ width: 240 }}
            value={ocr.apiBase}
          />
        </ProListItem>

        <ProListItem
          description={t("preference.ocr.api_settings.hints.api_key")}
          title={t("preference.ocr.api_settings.label.api_key")}
        >
          <Input.Password
            onChange={(e) => {
              globalStore.ocr.apiKey = e.target.value;
            }}
            placeholder="sk-..."
            style={{ width: 240 }}
            value={ocr.apiKey}
          />
        </ProListItem>

        <ProListItem
          description={t("preference.ocr.api_settings.hints.model")}
          title={t("preference.ocr.api_settings.label.model")}
        >
          <Input
            onChange={(e) => {
              globalStore.ocr.model = e.target.value;
            }}
            placeholder="gpt-4o"
            style={{ width: 240 }}
            value={ocr.model}
          />
        </ProListItem>

        <ProListItem
          description={t("preference.ocr.api_settings.hints.prompt")}
          title={t("preference.ocr.api_settings.label.prompt")}
        >
          <TextArea
            autoSize={{ maxRows: 4, minRows: 2 }}
            onChange={(e) => {
              globalStore.ocr.prompt = e.target.value;
            }}
            placeholder="请识别图片中的文字..."
            style={{ width: 300 }}
            value={ocr.prompt}
          />
        </ProListItem>
      </ProList>

      {/* 翻译设置 */}
      <ProList header={t("preference.ocr.translate_settings.title")}>
        <ProSwitch
          onChange={(value) => {
            globalStore.ocr.translate.enabled = value;
          }}
          title={t("preference.ocr.translate_settings.label.enabled")}
          value={ocr.translate.enabled}
        />

        <ProListItem
          description={t(
            "preference.ocr.translate_settings.hints.target_language",
          )}
          title={t("preference.ocr.translate_settings.label.target_language")}
        >
          <Select
            onChange={(value) => {
              globalStore.ocr.translate.targetLanguage = value;
            }}
            options={LANGUAGE_OPTIONS}
            style={{ width: 150 }}
            value={ocr.translate.targetLanguage}
          />
        </ProListItem>

        <ProListItem
          description={t("preference.ocr.translate_settings.hints.api_base")}
          title={t("preference.ocr.translate_settings.label.api_base")}
        >
          <Input
            onChange={(e) => {
              globalStore.ocr.translate.apiBase = e.target.value;
            }}
            placeholder={t(
              "preference.ocr.translate_settings.placeholder.api_base",
            )}
            style={{ width: 240 }}
            value={ocr.translate.apiBase}
          />
        </ProListItem>

        <ProListItem
          description={t("preference.ocr.translate_settings.hints.api_key")}
          title={t("preference.ocr.translate_settings.label.api_key")}
        >
          <Input.Password
            onChange={(e) => {
              globalStore.ocr.translate.apiKey = e.target.value;
            }}
            placeholder={t(
              "preference.ocr.translate_settings.placeholder.api_key",
            )}
            style={{ width: 240 }}
            value={ocr.translate.apiKey}
          />
        </ProListItem>

        <ProListItem
          description={t("preference.ocr.translate_settings.hints.model")}
          title={t("preference.ocr.translate_settings.label.model")}
        >
          <Input
            onChange={(e) => {
              globalStore.ocr.translate.model = e.target.value;
            }}
            placeholder={t(
              "preference.ocr.translate_settings.placeholder.model",
            )}
            style={{ width: 240 }}
            value={ocr.translate.model}
          />
        </ProListItem>

        <ProListItem
          description={t(
            "preference.ocr.translate_settings.hints.system_prompt",
          )}
          title={t("preference.ocr.translate_settings.label.system_prompt")}
        >
          <TextArea
            autoSize={{ maxRows: 4, minRows: 2 }}
            onChange={(e) => {
              globalStore.ocr.translate.systemPrompt = e.target.value;
            }}
            placeholder="You are a professional translator..."
            style={{ width: 300 }}
            value={ocr.translate.systemPrompt}
          />
        </ProListItem>
      </ProList>

      {/* 快捷键设置 */}
      <ProList header={t("preference.ocr.shortcut_settings.title")}>
        <ProShortcut
          description={t("preference.ocr.shortcut_settings.hints.ocr")}
          onChange={(value) => {
            globalStore.shortcut.ocr = value;
          }}
          title={t("preference.ocr.shortcut_settings.label.ocr")}
          value={shortcut.ocr}
        />

        <ProShortcut
          description={t(
            "preference.ocr.shortcut_settings.hints.ocr_translate",
          )}
          onChange={(value) => {
            globalStore.shortcut.ocrTranslate = value;
          }}
          title={t("preference.ocr.shortcut_settings.label.ocr_translate")}
          value={shortcut.ocrTranslate}
        />
      </ProList>

      {/* 行为设置 */}
      <ProList header={t("preference.ocr.behavior_settings.title")}>
        <ProSwitch
          description={t("preference.ocr.behavior_settings.hints.auto_copy")}
          onChange={(value) => {
            globalStore.ocr.autoCopy = value;
          }}
          title={t("preference.ocr.behavior_settings.label.auto_copy")}
          value={ocr.autoCopy}
        />

        <ProSwitch
          description={t(
            "preference.ocr.behavior_settings.hints.window_pinned",
          )}
          onChange={(value) => {
            globalStore.ocr.windowPinned = value;
          }}
          title={t("preference.ocr.behavior_settings.label.window_pinned")}
          value={ocr.windowPinned}
        />

        <ProSwitch
          description={t(
            "preference.ocr.behavior_settings.hints.hide_main_window",
          )}
          onChange={(value) => {
            globalStore.ocr.hideMainWindow = value;
          }}
          title={t("preference.ocr.behavior_settings.label.hide_main_window")}
          value={ocr.hideMainWindow}
        />

        <ProSwitch
          description={t("preference.ocr.behavior_settings.hints.save_history")}
          onChange={(value) => {
            globalStore.ocr.saveHistory = value;
          }}
          title={t("preference.ocr.behavior_settings.label.save_history")}
          value={ocr.saveHistory}
        />
      </ProList>
    </>
  );
};

export default OCRSettings;
