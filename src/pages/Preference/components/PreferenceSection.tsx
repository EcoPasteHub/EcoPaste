import { motion } from "motion/react";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import type { Settings } from "@/types/settings";
import type {
  PreferenceSection as PreferenceSectionModel,
  PreferenceSetting,
  SettingValue,
} from "../types/preferences";
import { translatePreferenceSection } from "../utils/preferenceI18n";
import PreferenceCountTag from "./PreferenceCountTag";
import PreferenceSettingRow from "./PreferenceSettingRow";
import SourceAppsTransfer from "./SourceAppsTransfer";

interface PreferenceSectionProps {
  highlightedSettingId: string | null;
  highlightToken: number;
  section: PreferenceSectionModel;
  settings: Settings;
  shouldReduceMotion: boolean;
  onChange: (setting: PreferenceSetting, value: SettingValue) => Promise<void>;
}

interface SectionVisual {
  icon: string;
}

/**
 * 偏好页主内容里的一个语义分组。
 */
const PreferenceSection: FC<PreferenceSectionProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const {
    highlightedSettingId,
    highlightToken,
    section,
    settings,
    shouldReduceMotion,
    onChange,
  } = props;
  const visual = resolveSectionVisual(section.id);
  const sourceAppsSettings = resolveSourceAppsSettings(section.settings);

  if (sourceAppsSettings) {
    return (
      <motion.section
        animate={{ opacity: 1 }}
        className="relative flex min-h-0 flex-1 scroll-mt-5 flex-col rounded-2 border border-ant-border-secondary bg-ant-container p-4"
        id={section.id}
        initial={{ opacity: 0 }}
        transition={{
          duration: shouldReduceMotion ? 0 : 0.12,
          ease: "easeOut",
        }}
      >
        <SourceAppsTransfer
          excludedAppsSetting={sourceAppsSettings.excludedApps}
          onChange={onChange}
          scanDirsSetting={sourceAppsSettings.scanDirs}
          settings={settings}
        />
      </motion.section>
    );
  }

  return (
    <motion.section
      animate={{ opacity: 1 }}
      className="relative scroll-mt-5 overflow-hidden rounded-2 border border-ant-border-secondary bg-ant-container"
      id={section.id}
      initial={{ opacity: 0 }}
      transition={{
        duration: shouldReduceMotion ? 0 : 0.12,
        ease: "easeOut",
      }}
    >
      <div className="relative flex items-center justify-between gap-4 border-ant-split border-b bg-ant-container px-4 py-3.5">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex size-9 shrink-0 items-center justify-center text-ant-primary text-xl">
            <i aria-hidden="true" className={visual.icon} />
          </span>

          <div className="min-w-0">
            <h2 className="m-0 truncate font-semibold text-ant-text text-sm leading-tight">
              {translatePreferenceSection(t, section, "title")}
            </h2>
          </div>
        </div>

        <PreferenceCountTag>
          {t("common:units.items", { count: section.settings.length })}
        </PreferenceCountTag>
      </div>

      <div>
        {section.settings.map((setting) => {
          return (
            <PreferenceSettingRow
              highlighted={setting.id === highlightedSettingId}
              highlightToken={highlightToken}
              key={setting.id}
              onChange={onChange}
              setting={setting}
              settings={settings}
              shouldReduceMotion={shouldReduceMotion}
            />
          );
        })}
      </div>
    </motion.section>
  );
};

export default PreferenceSection;

/**
 * 根据分组语义选择小图标，保持区块标题和设置行图标尺寸一致。
 */
function resolveSectionVisual(id: string): SectionVisual {
  if (id.includes("capture")) {
    return {
      icon: "i-lucide:clipboard-plus",
    };
  }

  if (id.includes("source")) {
    return {
      icon: "i-lucide:panels-top-left",
    };
  }

  if (id.includes("sensitive") || id.includes("diagnostics")) {
    return {
      icon: "i-lucide:shield-check",
    };
  }

  if (id.includes("history") || id.includes("localData")) {
    return {
      icon: "i-lucide:database",
    };
  }

  if (id.includes("organizing")) {
    return {
      icon: "i-lucide:star",
    };
  }

  if (id.includes("search")) {
    return {
      icon: "i-lucide:search",
    };
  }

  if (id.includes("paste") || id.includes("actions")) {
    return {
      icon: "i-lucide:mouse-pointer-click",
    };
  }

  if (id.includes("copy")) {
    return {
      icon: "i-lucide:copy",
    };
  }

  if (id.includes("window")) {
    return {
      icon: "i-lucide:panel-left",
    };
  }

  if (id.includes("preview")) {
    return {
      icon: "i-lucide:eye",
    };
  }

  if (id.includes("appearance")) {
    return {
      icon: "i-lucide:paintbrush",
    };
  }

  if (id.includes("control")) {
    return {
      icon: "i-lucide:monitor",
    };
  }

  if (id.includes("backup")) {
    return {
      icon: "i-lucide:refresh-cw",
    };
  }

  if (id.includes("external")) {
    return {
      icon: "i-lucide:plug",
    };
  }

  if (id.includes("updates")) {
    return {
      icon: "i-lucide:refresh-cw",
    };
  }

  return {
    icon: "i-lucide:clipboard-list",
  };
}

/**
 * 识别来源应用分组所需的两个设置项，缺任一项则退回通用行渲染。
 */
function resolveSourceAppsSettings(settings: PreferenceSetting[]) {
  const excludedApps = settings.find((setting) => {
    return setting.id === "source.excludedApps";
  });
  const scanDirs = settings.find((setting) => {
    return setting.id === "source.scanDirs";
  });

  if (!excludedApps || !scanDirs) return null;

  return { excludedApps, scanDirs };
}
