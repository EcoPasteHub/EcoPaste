import type { CheckboxGroupProps } from "antd";
import { Checkbox, Modal } from "antd";
import i18n from "@/i18n";

type ClearClipboardItemsChoice = "favorites" | "pinned";

export interface ClearClipboardItemsOptions {
  deleteFavorites: boolean;
  deletePinned: boolean;
}

/**
 * 弹出清空记录确认框，并返回是否连带删除收藏 / 置顶记录；取消时返回 null。
 */
export const confirmClearClipboardItems =
  async (): Promise<ClearClipboardItemsOptions | null> => {
    let choices: ClearClipboardItemsChoice[] = [];

    const handleChoicesChange: CheckboxGroupProps<ClearClipboardItemsChoice>["onChange"] =
      (nextChoices) => {
        choices = [...nextChoices];
      };

    return await new Promise<ClearClipboardItemsOptions | null>((resolve) => {
      Modal.confirm({
        cancelText: i18n.t("common:actions.cancel"),
        centered: true,
        content: (
          <div className="flex flex-col gap-3">
            <p className="m-0">{i18n.t("commands:clearConfirm.content")}</p>
            <Checkbox.Group
              className="flex flex-wrap gap-x-5 gap-y-2"
              defaultValue={[]}
              onChange={handleChoicesChange}
            >
              <Checkbox value="favorites">
                {i18n.t("commands:clearConfirm.deleteFavorites")}
              </Checkbox>
              <Checkbox value="pinned">
                {i18n.t("commands:clearConfirm.deletePinned")}
              </Checkbox>
            </Checkbox.Group>
          </div>
        ),
        okButtonProps: { danger: true },
        okText: i18n.t("common:actions.clear"),
        onCancel: () => {
          resolve(null);
        },
        onOk: () => {
          resolve({
            deleteFavorites: choices.includes("favorites"),
            deletePinned: choices.includes("pinned"),
          });
        },
        title: i18n.t("commands:clearConfirm.title"),
      });
    });
  };
