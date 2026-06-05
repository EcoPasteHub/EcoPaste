import { Empty } from "antd";
import { motion } from "motion/react";
import type { FC } from "react";
import type { allPreferenceSettings } from "../config/preferenceSchema";

type SearchResult = (typeof allPreferenceSettings)[number];

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

  if (!query) return null;

  return (
    <motion.div
      animate={{ opacity: 1, y: 0 }}
      className="absolute top-full right-0 left-0 z-10 mt-2 max-h-88 overflow-y-auto overscroll-contain rounded-2 border border-ant-border-secondary bg-ant-elevated shadow-sm"
      initial={{ opacity: 0, y: shouldReduceMotion ? 0 : -4 }}
      transition={{ duration: shouldReduceMotion ? 0 : 0.16, ease: "easeOut" }}
    >
      {results.length === 0 ? (
        <div className="p-4">
          <Empty
            description="没有找到相关设置"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      ) : (
        results.map((result) => {
          const handleClick = () => {
            onPick(result);
          };

          return (
            <button
              className="flex w-full cursor-pointer items-center gap-3 border-0 border-ant-split border-b bg-transparent px-3.5 py-3 text-left text-ant-text transition-colors last:border-b-0 hover:bg-ant-fill-tertiary focus-visible:bg-ant-fill-tertiary motion-reduce:transition-none"
              key={result.setting.id}
              onClick={handleClick}
              type="button"
            >
              <div className="min-w-0 flex-1">
                <span className="block truncate font-medium text-sm">
                  {result.setting.title}
                </span>
                <small className="mt-0.5 block truncate text-ant-secondary text-xs leading-snug">
                  {result.setting.description}
                </small>
              </div>
              <div className="whitespace-nowrap text-ant-tertiary text-xs">
                {result.tab.title} / {result.section.title}
              </div>
            </button>
          );
        })
      )}
    </motion.div>
  );
};

export default PreferenceSearchResults;
