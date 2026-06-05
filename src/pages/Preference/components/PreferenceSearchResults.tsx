import { Empty } from "antd";
import { AnimatePresence, motion } from "motion/react";
import { type FC, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { allPreferenceSettings } from "../config/preferenceSchema";
import {
  translatePreferenceSection,
  translatePreferenceSetting,
  translatePreferenceTab,
} from "../utils/preferenceI18n";

type SearchResult = (typeof allPreferenceSettings)[number];

const PANEL_MIN_WIDTH = "16rem";

interface PreferenceSearchResultsProps {
  shouldReduceMotion: boolean;
  query: string;
  results: SearchResult[];
  onPick: (result: SearchResult) => void;
}

/**
 * 全局设置搜索结果：直接跳转到设置所在位置。
 */
const PreferenceSearchResults: FC<PreferenceSearchResultsProps> = (props) => {
  const { query, results, shouldReduceMotion, onPick } = props;
  const { t } = useTranslation("preferences");
  const contentRef = useRef<HTMLDivElement | null>(null);
  const [panelWidth, setPanelWidth] = useState(PANEL_MIN_WIDTH);
  const [panelReady, setPanelReady] = useState(false);
  const offsetY = shouldReduceMotion ? 0 : -4;
  const transition = {
    duration: shouldReduceMotion ? 0 : 0.16,
    ease: "easeOut",
  } as const;
  const layoutTransition = {
    duration: shouldReduceMotion ? 0 : 0.18,
    ease: "easeOut",
  } as const;

  useLayoutEffect(() => {
    if (!query) {
      setPanelReady(false);
      return;
    }

    const node = contentRef.current;
    if (!node) return;

    const updatePanelWidth = () => {
      const rootFontSize =
        Number.parseFloat(
          window.getComputedStyle(document.documentElement).fontSize,
        ) || 16;
      const nextWidth = node.getBoundingClientRect().width / rootFontSize;

      setPanelWidth(`${nextWidth}rem`);
      setPanelReady(true);
    };

    updatePanelWidth();

    const observer = new ResizeObserver(updatePanelWidth);
    observer.observe(node);

    return () => {
      observer.disconnect();
    };
  }, [query]);

  return (
    <AnimatePresence>
      {query ? (
        <motion.div
          animate={{ opacity: panelReady ? 1 : 0, width: panelWidth, y: 0 }}
          className="absolute top-full right-0 z-10 mt-2 overflow-hidden rounded-2 border border-ant-border-secondary bg-ant-elevated shadow-sm"
          exit={{ opacity: 0, width: panelWidth, y: offsetY }}
          initial={{ opacity: 0, width: panelWidth, y: offsetY }}
          style={{ pointerEvents: panelReady ? "auto" : "none" }}
          transition={{ ...transition, width: layoutTransition }}
        >
          <AnimatePresence initial={false} mode="popLayout">
            {results.length === 0 ? (
              <motion.div
                animate={{ opacity: 1 }}
                className="w-64 p-4"
                exit={{ opacity: 0 }}
                initial={{ opacity: 0 }}
                key="empty"
                ref={contentRef}
                transition={transition}
              >
                <Empty
                  description={t("search.empty")}
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              </motion.div>
            ) : (
              <motion.div
                animate={{ opacity: 1 }}
                className="max-h-88 w-max min-w-64 max-w-128 overflow-y-auto overscroll-contain"
                exit={{ opacity: 0 }}
                initial={{ opacity: 0 }}
                key="results"
                ref={contentRef}
                transition={transition}
              >
                {results.map((result) => {
                  const handleClick = () => {
                    onPick(result);
                  };

                  return (
                    <button
                      className="flex w-full min-w-0 cursor-pointer items-center gap-3 border-0 border-ant-split border-b bg-transparent px-3.5 py-3 text-left text-ant-text transition-colors last:border-b-0 hover:bg-ant-fill-tertiary focus-visible:bg-ant-fill-tertiary motion-reduce:transition-none"
                      key={result.setting.id}
                      onClick={handleClick}
                      type="button"
                    >
                      <div className="min-w-0 flex-1">
                        <span className="block truncate font-medium text-sm">
                          {translatePreferenceSetting(
                            t,
                            result.setting,
                            "title",
                          )}
                        </span>
                        <small className="mt-0.5 block truncate text-ant-secondary text-xs leading-snug">
                          {translatePreferenceSetting(
                            t,
                            result.setting,
                            "description",
                          )}
                        </small>
                      </div>
                      <div className="max-w-56 truncate text-ant-tertiary text-xs">
                        {translatePreferenceTab(t, result.tab)} /{" "}
                        {translatePreferenceSection(t, result.section, "title")}
                      </div>
                    </button>
                  );
                })}
              </motion.div>
            )}
          </AnimatePresence>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
};

export default PreferenceSearchResults;
