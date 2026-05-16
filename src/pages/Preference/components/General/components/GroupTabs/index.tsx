import { useBoolean, useCreation } from "ahooks";
import { Button, Flex, Modal, Switch, Tree, type TreeProps } from "antd";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProListItem from "@/components/ProListItem";
import UnoIcon from "@/components/UnoIcon";
import { globalStore } from "@/stores/global";
import type { AppearanceGroupTab } from "@/types/store";
import {
  DEFAULT_APPEARANCE_GROUP_TABS,
  GROUP_LABEL_KEYS,
  normalizeAppearanceGroupTabs,
} from "@/utils/group";

const GroupTabs = () => {
  const { appearance } = useSnapshot(globalStore);
  const [open, { toggle }] = useBoolean();
  const { t } = useTranslation();

  const treeData = useCreation(() => {
    return normalizeAppearanceGroupTabs(appearance.groupTabs).map((item) => ({
      ...item,
      key: item.id,
    }));
  }, [appearance.groupTabs]);

  const updateVisible = (id: AppearanceGroupTab["id"], visible: boolean) => {
    const tabs = normalizeAppearanceGroupTabs(globalStore.appearance.groupTabs);
    const target = tabs.find((item) => item.id === id);

    if (!target) return;

    target.visible = visible;
    globalStore.appearance.groupTabs = tabs;
  };

  const resetGroupTabs = () => {
    globalStore.appearance.groupTabs = DEFAULT_APPEARANCE_GROUP_TABS.map(
      (item) => ({ ...item }),
    );
  };

  const handleDrop: TreeProps["onDrop"] = (info) => {
    const { dragNode, node, dropPosition } = info;
    const getIndex = (pos: string) => pos.split("-").map(Number)[1];
    const dragIndex = getIndex(dragNode.pos);
    let dropIndex = getIndex(node.pos);
    const tabs = normalizeAppearanceGroupTabs(globalStore.appearance.groupTabs);

    if (dragIndex > dropIndex && dropPosition > 0) {
      dropIndex++;
    }

    tabs.splice(dropIndex, 0, ...tabs.splice(dragIndex, 1));
    globalStore.appearance.groupTabs = tabs;
  };

  return (
    <>
      <ProListItem
        description={t("preference.settings.appearance_settings.hints.group_tabs")}
        title={t("preference.settings.appearance_settings.label.group_tabs")}
      >
        <Button onClick={toggle}>
          {t("preference.settings.appearance_settings.button.custom_group_tabs")}
        </Button>
      </ProListItem>

      <Modal
        centered
        destroyOnClose
        footer={null}
        onCancel={toggle}
        open={open}
        title={t(
          "preference.settings.appearance_settings.label.custom_group_tabs_title",
        )}
        width={480}
      >
        <Flex className="pb-3 text-color-2 text-sm" justify="space-between">
          <span>{t(GROUP_LABEL_KEYS.all)}</span>
          <span>
            {t("preference.settings.appearance_settings.hints.group_tabs_fixed")}
          </span>
        </Flex>

        <Flex className="pb-3" justify="space-between">
          <span className="text-color-2 text-sm">
            {t("preference.settings.appearance_settings.hints.group_tabs_drag")}
          </span>

          <Button onClick={resetGroupTabs} size="small">
            {t("preference.settings.appearance_settings.button.reset_group_tabs")}
          </Button>
        </Flex>

        <Tree
          blockNode
          className="[&_.ant-tree-switcher]:hidden"
          draggable
          onDrop={handleDrop}
          selectable={false}
          titleRender={(item) => {
            const data = item as AppearanceGroupTab;

            return (
              <Flex
                align="center"
                className="w-full cursor-move"
                justify="space-between"
                onClick={(event) => {
                  event.stopPropagation();
                }}
              >
                <Flex align="center" gap={8}>
                  <UnoIcon
                    className="cursor-grab text-color-3"
                    name="i-lucide:grip-vertical"
                  />
                  <span>{t(GROUP_LABEL_KEYS[data.id])}</span>
                </Flex>

                <div
                  onClick={(event) => {
                    event.stopPropagation();
                  }}
                  onMouseDown={(event) => {
                    event.stopPropagation();
                  }}
                >
                  <Switch
                    checked={data.visible}
                    onChange={(value) => {
                      updateVisible(data.id, value);
                    }}
                    size="small"
                  />
                </div>
              </Flex>
            );
          }}
          treeData={treeData}
        />
      </Modal>
    </>
  );
};

export default GroupTabs;
