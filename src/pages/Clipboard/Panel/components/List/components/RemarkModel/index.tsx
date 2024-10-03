import { ClipboardPanelContext } from "@/pages/Clipboard/Panel";
import type { ClipboardItem } from "@/types/database";
import { Form, Input, Modal } from "antd";
import { find } from "lodash-es";

export interface RemarkModalRef {
	open: () => void;
}

interface FormFields {
	remark: string;
}

const RemarkModal = forwardRef<RemarkModalRef>((_, ref) => {
	const { state } = useContext(ClipboardPanelContext);
	const [open, { toggle }] = useBoolean();
	const [item, setItem] = useState<ClipboardItem>();
	const [form] = Form.useForm<FormFields>();

	useImperativeHandle(ref, () => ({
		open: () => {
			const findItem = find(state.list, { id: state.activeId });

			form.setFieldsValue({
				remark: findItem?.remark,
			});

			setItem(findItem);

			toggle();
		},
	}));

	const handleOk = () => {
		const { remark } = form.getFieldsValue();

		if (item) {
			item.remark = remark;

			updateSQL("history", { id: item.id, remark });
		}

		toggle();
	};

	return (
		<Modal
			forceRender
			centered
			title="备注"
			open={open}
			onOk={handleOk}
			onCancel={toggle}
		>
			<Form
				form={form}
				initialValues={{ remark: item?.remark }}
				onFinish={handleOk}
			>
				<Form.Item name="remark" className="mb-0!">
					<Input placeholder="请输入备注" />
				</Form.Item>
			</Form>
		</Modal>
	);
});

export default RemarkModal;
