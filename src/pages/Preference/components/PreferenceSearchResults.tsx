import { Empty } from "antd";
import { AnimatePresence, motion } from "motion/react";
import { type FC, useLayoutEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import ScrollArea from "@/components/ScrollArea";
import type { allPreferenceSettings } from "../config/preferenceSchema";
import {
  translatePreferenceSection,
  translatePreferenceSetting,
  translatePreferenceTab,
} from "../utils/preferenceI18n";

type SearchResult = (typeof allPreferenceSettings)[number];

const PANEL_MIN_WIDTH = "16rem";
const PANEL_MIN_HEIGHT = "0rem";

interface PanelSize {
  width: string;
  height: string;
}

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
  const [measureNode, setMeasureNode] = useState<HTMLDivElement | null>(null);
  const [panelSize, setPanelSize] = useState<PanelSize>({
    height: PANEL_MIN_HEIGHT,
    width: PANEL_MIN_WIDTH,
  });
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
  const panelContentKey =
    results.length === 0
      ? "empty"
      : results
          .map((result) => {
            return result.setting.id;
          })
          .join("|");

  useLayoutEffect(() => {
    if (!query) {
      setPanelReady(false);
      return;
    }

    const node = measureNode;
    if (!node) return;
    if (node.dataset.measureKey !== panelContentKey) return;

    const updatePanelSize = () => {
      const rootFontSize =
        Number.parseFloat(
          window.getComputedStyle(document.documentElement).fontSize,
        ) || 16;
      const rect = node.getBoundingClientRect();
      const nextWidth = rect.width / rootFontSize;
      const nextHeight = rect.height / rootFontSize;

      setPanelSize({
        height: `${nextHeight}rem`,
        width: `${nextWidth}rem`,
      });
      setPanelReady(true);
    };

    updatePanelSize();

    const observer = new ResizeObserver(updatePanelSize);
    observer.observe(node);

    return () => {
      observer.disconnect();
    };
  }, [query, panelContentKey, measureNode]);

  /**
   * 渲染搜索结果内容；隐藏测量层复用同一结构，保证动画目标尺寸和可见内容一致。
   */
  const renderPanelContent = () => {
    if (results.length === 0) {
      return (
        <div className="w-64 overflow-hidden p-4">
          <Empty
            description={t("search.empty")}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      );
    }

    return (
      <ScrollArea className="max-h-88 w-max min-w-64 max-w-128 overflow-x-hidden overscroll-contain">
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
                  {translatePreferenceSetting(t, result.setting, "title")}
                </span>
                <small className="mt-0.5 block truncate text-ant-secondary text-xs leading-snug">
                  {translatePreferenceSetting(t, result.setting, "description")}
                </small>
              </div>
              <div className="max-w-56 truncate text-ant-tertiary text-xs">
                {translatePreferenceTab(t, result.tab)} /{" "}
                {translatePreferenceSection(t, result.section, "title")}
              </div>
            </button>
          );
        })}
      </ScrollArea>
    );
  };

  return (
    <>
      {query ? (
        <div
          className="pointer-events-none invisible absolute top-full right-0 z-0 mt-2"
          data-measure-key={panelContentKey}
          ref={setMeasureNode}
        >
          {renderPanelContent()}
        </div>
      ) : null}

      <AnimatePresence>
        {query ? (
          <motion.div
            animate={{
              height: panelSize.height,
              opacity: panelReady ? 1 : 0,
              width: panelSize.width,
              y: 0,
            }}
            className="absolute top-full right-0 z-10 mt-2 overflow-hidden rounded-2 border border-ant-border-secondary bg-ant-elevated shadow-sm"
            exit={{
              height: panelSize.height,
              opacity: 0,
              width: panelSize.width,
              y: offsetY,
            }}
            initial={{
              height: panelSize.height,
              opacity: 0,
              width: panelSize.width,
              y: offsetY,
            }}
            style={{ pointerEvents: panelReady ? "auto" : "none" }}
            transition={{
              ...transition,
              height: layoutTransition,
              width: layoutTransition,
            }}
          >
            <AnimatePresence initial={false} mode="popLayout">
              <motion.div
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                initial={{ opacity: 0 }}
                key={panelContentKey}
                transition={transition}
              >
                {renderPanelContent()}
              </motion.div>
            </AnimatePresence>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </>
  );
};

export default PreferenceSearchResults;
