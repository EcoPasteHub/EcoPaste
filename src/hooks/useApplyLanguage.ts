import { useEffect } from "react";
import { useSnapshot } from "valtio";
import i18n from "@/locales";
import { settingsState } from "@/stores/settings";
import { log } from "@/utils/log";

/**
 * 跟随 settings.appearance.language 调用 i18next.changeLanguage。
 * 初始语言已在 initI18n 阶段设过，这里只处理运行时切换。
 */
export function useApplyLanguage(): void {
  const { value, loaded } = useSnapshot(settingsState);
  const language = value?.appearance.language;

  useEffect(() => {
    if (!loaded || !language || i18n.language === language) return;
    i18n.changeLanguage(language).catch((err) => {
      log.error("change language failed", err);
    });
  }, [language, loaded]);
}
