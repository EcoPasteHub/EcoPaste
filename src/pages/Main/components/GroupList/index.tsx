import { useKeyPress } from "ahooks";
import { Tag } from "antd";
import clsx from "clsx";
import { useContext, useEffect } from "react";
import { useTranslation } from "react-i18next";
import Scrollbar from "@/components/Scrollbar";
import type { DatabaseSchemaGroup } from "@/types/database";
import { scrollElementToCenter } from "@/utils/dom";
import { MainContext } from "../..";

const GroupList = () => {
  const { rootState } = useContext(MainContext);
  const { t } = useTranslation();

  useEffect(() => {
    scrollElementToCenter(rootState.group);
  }, [rootState.group]);

  const presetGroups: DatabaseSchemaGroup[] = [
    {
      id: "all",
      name: t("clipboard.label.tab.all"),
    },
    {
      id: "text",
      name: t("clipboard.label.tab.text"),
    },
    {
      id: "image",
      name: t("clipboard.label.tab.image"),
    },
    {
      id: "files",
      name: t("clipboard.label.tab.files"),
    },
    {
      id: "favorite",
      name: t("clipboard.label.tab.favorite"),
    },
  ];

  useKeyPress("tab", (event) => {
    const index = presetGroups.findIndex((item) => item.id === rootState.group);
    const length = presetGroups.length;

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
