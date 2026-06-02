import { Input, Modal } from "antd";
import type { ChangeEvent, FC } from "react";
import { useEffect, useState } from "react";
import { updateClipboardItemNote } from "@/commands";
import type { ClipboardItem } from "@/types/clipboard";

interface NoteModalProps {
  /**
   * 当前编辑目标；为 null 时关闭。由列表层持有的单例状态注入，引用稳定，
   * 仅在打开新目标时变化（据此重置输入框）。
   */
  item: ClipboardItem | null;
  /**
   * 关闭弹窗（取消或保存成功后调用）。
   */
  onClose: () => void;
  /**
   * 保存成功回调：归一化后的备注（空串视为清空，传 null）与后端是否触发了
   * auto-favorite，供列表层同步本地镜像的 `note` / `isFavorite`。
   */
  onSaved: (id: string, note: string | null, autoFavorited: boolean) => void;
}

/**
 * 备注编辑弹窗（列表层单例）。确定时调用 `update_clipboard_item_note`，
 * 后端把空串归一化为 NULL，并按 `autoFavorite` 设置联动收藏状态。
 */
const NoteModal: FC<NoteModalProps> = (props) => {
  const { item, onClose, onSaved } = props;

  const [value, setValue] = useState("");
  const [saving, setSaving] = useState(false);

  // item 引用稳定，仅在打开新目标时变化：据此把输入框重置为该条已有备注。
  useEffect(() => {
    if (!item) return;

    setValue(item.note ?? "");
  }, [item]);

  const handleChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setValue(event.target.value);
  };

  const handleSave = async () => {
    if (!item) return;

    setSaving(true);

    try {
      const { note, autoFavorited } = await updateClipboardItemNote(
        item.id,
        value,
      );

      onSaved(item.id, note, autoFavorited);
      onClose();
    } catch {
      // 错误 toast 由 commands 层统一处理。
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      confirmLoading={saving}
      destroyOnHidden
      okText="保存"
      onCancel={onClose}
      onOk={handleSave}
      open={!!item}
      title="编辑备注"
    >
      <Input.TextArea
        autoFocus
        autoSize={{ maxRows: 6, minRows: 3 }}
        maxLength={256}
        onChange={handleChange}
        placeholder="为这条记录添加备注..."
        showCount
        value={value}
      />
    </Modal>
  );
};

export default NoteModal;
