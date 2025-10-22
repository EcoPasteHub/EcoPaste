import { Form, Input, type InputRef, Modal } from "antd";
import { find } from "es-toolkit/compat";
import { t } from "i18next";
import { MainContext } from "@/pages/Main";
import type { DatabaseSchemaHistory } from "@/types/database";

export interface NoteModalRef {
  open: () => void;
}

interface FormFields {
  note: string;
}

const NoteModal = forwardRef<NoteModalRef>((_, ref) => {
  const { rootState } = useContext(MainContext);
  const [open, { toggle }] = useBoolean();
  const [item, setItem] = useState<DatabaseSchemaHistory>();
  const [form] = Form.useForm<FormFields>();
  const inputRef = useRef<InputRef>(null);

  useImperativeHandle(ref, () => ({
    open: () => {
      const findItem = find(rootState.list, { id: rootState.activeId });

      form.setFieldsValue({
        note: findItem?.note,
      });

      setItem(findItem);

      toggle();
    },
  }));

  const handleOk = async () => {
    const { note } = form.getFieldsValue();

    if (item) {
      const { id, favorite } = item;

      item.note = note;

      updateHistory(id, { note });

      if (clipboardStore.content.autoFavorite && !favorite) {
        item.favorite = true;

        updateHistory(id, { favorite: true });
      }
    }

    toggle();
  };

  const handleAfterOpenChange = (open: boolean) => {
    if (!open) return;

    inputRef.current?.focus();
  };

  return (
    <Modal
      afterOpenChange={handleAfterOpenChange}
      centered
      forceRender
      onCancel={toggle}
      onOk={handleOk}
      open={open}
      title={t("component.note_modal.label.note")}
    >
      <Form
        form={form}
        initialValues={{ note: item?.note }}
        onFinish={handleOk}
      >
        <Form.Item className="mb-0!" name="note">
          <Input
            autoComplete="off"
            placeholder={t("component.note_modal.hints.input_note")}
            ref={inputRef}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
});

export default NoteModal;
