import { Input } from "antd";
import type { ChangeEvent, FC } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/utils/cn";
import { PREFERENCE_TAB_META } from "../constants";
import type { PreferenceSection, PreferenceTab } from "../types/preferences";
import {
  translatePreferenceSection,
  translatePreferenceTab,
} from "../utils/preferenceI18n";
import type { PreferenceSearchResult } from "../utils/preferenceSearch";
import PreferenceCountTag from "./PreferenceCountTag";
import PreferenceSearchResults from "./PreferenceSearchResults";

interface PreferenceHeaderProps {
  activeSectionId: string;
  activeTab: PreferenceTab;
  isAboutTab: boolean;
  searchQuery: string;
  searchResults: PreferenceSearchResult[];
  shouldReduceMotion: boolean;
  totalSettings: number;
  onPickSearchResult: (result: PreferenceSearchResult) => void;
  onSearchChange: (event: ChangeEvent<HTMLInputElement>) => void;
  onSectionSelect: (sectionId: string) => void;
}

/**
 * 偏好窗口主区域头部：标题、全局搜索和二级分组导航。
 */
const PreferenceHeader: FC<PreferenceHeaderProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const {
    activeSectionId,
    activeTab,
    isAboutTab,
    searchQuery,
    searchResults,
    shouldReduceMotion,
    totalSettings,
    onPickSearchResult,
    onSearchChange,
    onSectionSelect,
  } = props;

  return (
    <header
      className={cn(
        "shrink-0 border-ant-border-secondary border-b bg-ant-container px-6 pt-4",
        isAboutTab ? "pb-4" : "pb-2",
      )}
      data-tauri-drag-region
    >
      <div
        className="flex items-center justify-between gap-5"
        data-tauri-drag-region
      >
        <div className="min-w-0">
          <h1 className="m-0 flex items-center gap-2 font-semibold text-ant-text text-lg leading-snug">
            <i
              aria-hidden="true"
              className={cn(
                "text-ant-primary text-lg",
                PREFERENCE_TAB_META[activeTab.id].icon,
              )}
            />
            <span className="truncate">
              {translatePreferenceTab(t, activeTab)}
            </span>
          </h1>
        </div>

        <div className="flex shrink-0 items-center">
          <div className="relative z-3 w-64">
            <Input
              allowClear
              autoCapitalize="off"
              autoCorrect="off"
              className="border-ant-border-secondary bg-ant-fill-quaternary text-ant-text"
              onChange={onSearchChange}
              placeholder={t("preferences:search.placeholder")}
              prefix={
                <i
                  aria-hidden="true"
                  className="i-lucide:search text-ant-secondary text-base"
                />
              }
              spellCheck={false}
              value={searchQuery}
            />

            <PreferenceSearchResults
              onPick={onPickSearchResult}
              query={searchQuery.trim()}
              results={searchResults}
              shouldReduceMotion={shouldReduceMotion}
            />
          </div>
        </div>
      </div>

      {!isAboutTab ? (
        <SectionTabs
          activeSectionId={activeSectionId}
          onSectionSelect={onSectionSelect}
          sections={activeTab.sections}
          totalSettings={totalSettings}
        />
      ) : null}
    </header>
  );
};

interface SectionTabsProps {
  activeSectionId: string;
  sections: PreferenceSection[];
  totalSettings: number;
  onSectionSelect: (sectionId: string) => void;
}

/**
 * 偏好页二级分组导航，紧贴标题栏用于快速切换当前分类。
 */
const SectionTabs: FC<SectionTabsProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { activeSectionId, sections, totalSettings, onSectionSelect } = props;

  return (
    <div className="mt-3 flex items-center gap-3" data-tauri-drag-region>
      {sections.map((section) => {
        const selected = section.id === activeSectionId;
        const handleClick = () => {
          onSectionSelect(section.id);
        };

        return (
          <button
            className={cn(
              "relative h-7.5 cursor-pointer whitespace-nowrap border-0 bg-transparent px-0.5 font-medium text-sm transition-colors focus-visible:ring-1 focus-visible:ring-ant-primary motion-reduce:transition-none",
              selected
                ? "text-ant-text"
                : "text-ant-secondary hover:text-ant-text",
            )}
            key={section.id}
            onClick={handleClick}
            type="button"
          >
            {translatePreferenceSection(t, section, "title")}
            <span
              className={cn(
                "absolute right-0 bottom-0 left-0 h-0.5 rounded-full transition-colors motion-reduce:transition-none",
                selected ? "bg-ant-primary" : "bg-transparent",
              )}
            />
          </button>
        );
      })}

      <PreferenceCountTag className="ml-auto">
        {t("common:units.settings", { count: totalSettings })}
      </PreferenceCountTag>
    </div>
  );
};

export default PreferenceHeader;
