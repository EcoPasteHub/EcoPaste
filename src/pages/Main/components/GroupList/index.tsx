import { useKeyPress } from "ahooks";
import { Tag } from "antd";
import clsx from "clsx";
import { useContext, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import Scrollbar from "@/components/Scrollbar";
import type { DatabaseSchemaGroup } from "@/types/database";
import { globalStore } from "@/stores/global";
import { scrollElementToCenter } from "@/utils/dom";
import { getVisibleClipboardGroups, GROUP_LABEL_KEYS } from "@/utils/group";
import { MainContext } from "../..";

const GroupList = () => {
  const { rootState } = useContext(MainContext);
  const { appearance } = useSnapshot(globalStore);
  const { t } = useTranslation();
  const visibleGroupIds = getVisibleClipboardGroups(appearance.groupTabs);
  const visibleGroupSignature = visibleGroupIds.join(",");
  const presetGroups: DatabaseSchemaGroup[] = visibleGroupIds.map((id) => {
    return {
      id,
      name: t(GROUP_LABEL_KEYS[id]),
    };
  });

  useEffect(() => {
    scrollElementToCenter(rootState.group);
  }, [rootState.group]);

  useEffect(() => {
    if (visibleGroupIds.some((id) => id === rootState.group)) return;

    rootState.group = "all";
  }, [rootState.group, visibleGroupSignature]);

  useKeyPress("tab", (event) => {
    const index = presetGroups.findIndex((item) => item.id === rootState.group);
    const length = presetGroups.length;

    if (index < 0 || length === 0) {
      rootState.group = "all";

      return;
    }

    let nextIndex = index;

    if (event.shiftKey) {
      nextIndex = index === 0 ? length - 1 : index - 1;
    } else {
      nextIndex = index === length - 1 ? 0 : index + 1;
    }

    rootState.group = presetGroups[nextIndex].id;
  });

  return (
    <Scrollbar className="flex" data-tauri-drag-region>
      {presetGroups.map((item) => {
        const { id, name } = item;

        const isChecked = id === rootState.group;

        return (
          <div id={id} key={id}>
            <Tag.CheckableTag
              checked={isChecked}
              className={clsx({ "bg-primary!": isChecked })}
              onChange={() => {
                rootState.group = id;
              }}
            >
              {name}
            </Tag.CheckableTag>
          </div>
        );
      })}
    </Scrollbar>
  );
};

export default GroupList;
