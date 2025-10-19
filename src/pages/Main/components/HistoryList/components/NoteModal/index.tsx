import { MainContext } from "@/pages/Main";
import type { DatabaseSchemaHistory } from "@/types/database";
import { Form, Input, type InputRef, Modal } from "antd";
import { find } from "es-toolkit/compat";
import { t } from "i18next";

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

			const db = await getDatabase();

			db.updateTable("history")
				.set("note", note)
				.where("id", "=", id)
				.execute();

			if (clipboardStore.content.autoFavorite && !favorite) {
				item.favorite = true;

				db.updateTable("history")
					.set("favorite", true)
					.where("id", "=", id)
					.execute();
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
