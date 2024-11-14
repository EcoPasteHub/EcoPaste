import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import type { HistoryTablePayload } from "@/types/database";
import { Form, Input, type InputRef, Modal } from "antd";
import { t } from "i18next";
import { find } from "lodash-es";

export interface NoteModalRef {
	open: () => void;
}

interface FormFields {
	note: string;
}

const NoteModal = forwardRef<NoteModalRef>((_, ref) => {
	const { state } = useContext(ClipboardPanelContext);
	const [open, { toggle }] = useBoolean();
	const [item, setItem] = useState<HistoryTablePayload>();
	const [form] = Form.useForm<FormFields>();
	const inputRef = useRef<InputRef>(null);

	useImperativeHandle(ref, () => ({
		open: () => {
			const findItem = find(state.list, { id: state.activeId });

			form.setFieldsValue({
				note: findItem?.note,
			});

			setItem(findItem);

			toggle();
		},
	}));

	const handleOk = () => {
		const { note } = form.getFieldsValue();

		if (item) {
			item.note = note;

			updateSQL("history", { id: item.id, note });
		}

		toggle();
	};

	const handleAfterOpenChange = (open: boolean) => {
		if (!open) return;

		inputRef.current?.focus();
	};

	return (
		<Modal
			forceRender
			centered
			title={t("component.note_modal.label.note")}
			open={open}
			onOk={handleOk}
			onCancel={toggle}
			afterOpenChange={handleAfterOpenChange}
		>
			<Form
				form={form}
				initialValues={{ note: item?.note }}
				onFinish={handleOk}
			>
				<Form.Item name="note" className="mb-0!">
					<Input
						ref={inputRef}
						autoComplete="off"
						placeholder={t("component.note_modal.hints.input_note")}
					/>
				</Form.Item>
			</Form>
		</Modal>
	);
});

export default NoteModal;
