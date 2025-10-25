import { Flex } from "antd";
import clsx from "clsx";
import { useSnapshot } from "valtio";
import UnoIcon from "@/components/UnoIcon";
import { showWindow } from "@/plugins/window";
import { clipboardStore } from "@/stores/clipboard";
import { isLinux, isWin } from "@/utils/is";
import GroupList from "../GroupList";
import HistoryList from "../HistoryList";
import SearchInput from "../SearchInput";
import WindowPin from "../WindowPin";

const StandardMode = () => {
  const { search } = useSnapshot(clipboardStore);

  return (
    <Flex
      className={clsx("h-screen bg-color-1 py-3", {
        "b b-color-1": isLinux,
        "flex-col-reverse": search.position === "bottom",
        "rounded-2.5": !isWin,
      })}
      data-tauri-drag-region
      gap={12}
      vertical
    >
      <SearchInput className="mx-3" />

      <Flex
        className="flex-1 overflow-hidden"
        data-tauri-drag-region
        gap={12}
        vertical
      >
        <Flex
          align="center"
          className="overflow-hidden px-3"
          data-tauri-drag-region
          gap="small"
          justify="space-between"
        >
          <GroupList />

          <Flex align="center" className="text-color-2 text-lg" gap={4}>
            <WindowPin />

            <UnoIcon
              hoverable
              name="i-lets-icons:setting-alt-line"
              onClick={() => {
                showWindow("preference");
              }}
            />
          </Flex>
        </Flex>

        <HistoryList />
      </Flex>
    </Flex>
  );
};

export default StandardMode;
