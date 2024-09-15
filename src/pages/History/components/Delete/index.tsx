import EcoSelect from "@/components/EcoSelect";
import type { ClipboardItem } from "@/types/database";
import { DeleteOutlined } from "@ant-design/icons";
import { emit } from "@tauri-apps/api/event";
import {
	Button,
	Checkbox,
	DatePicker,
	Drawer,
	Form,
	Space,
	message,
} from "antd";
import type { DefaultOptionType } from "antd/es/select";
import type { Dayjs } from "dayjs";

const { RangePicker } = DatePicker;

interface FormFields {
	timeRange: number;
	customRange: Dayjs[];
	deleteFavorite: boolean;
}

const Delete = () => {
	const [open, { toggle }] = useBoolean();
	const [form] = Form.useForm<FormFields>();
	const timeRange = Form.useWatch("timeRange", form);
	const [deleting, { setTrue, setFalse }] = useBoolean();

	useEffect(form.resetFields, [open]);

	const rangeOptions: DefaultOptionType[] = [
		{
			label: "过去 1 小时",
			value: 1,
		},
		{
			label: "过去 1 天",
			value: 24,
		},
		{
			label: "过去 7 天",
			value: 7 * 24,
		},
		{
			label: "过去 30 天",
			value: 30 * 24,
		},
		{
			label: "时间不限",
			value: 0,
		},
		{
			label: "自定义",
			value: -1,
		},
	];

	const onSubmit = async () => {
		const { timeRange, customRange, deleteFavorite } = form.getFieldsValue();

		setTrue();

		let range: Dayjs[] = [];

		if (timeRange < 0) {
			range = customRange;
		} else {
			range = [dayjs().subtract(timeRange, "hour"), dayjs()];
		}

		const formatRange = range.map((item) => item.format("YYYY-MM-DD HH:mm:ss"));

		const list = await selectSQL<ClipboardItem[]>("history");

		for await (const item of list) {
			const { id, favorite, createTime } = item;

			if (favorite && !deleteFavorite) continue;

			const isBetween = dayjs(createTime).isBetween(
				formatRange[0],
				formatRange[1],
				null,
				"[]",
			);

			if (timeRange === 0 || isBetween) {
				await deleteSQL("history", id);
			}
		}

		toggle();
		setFalse();
		message.success("删除成功");
		emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);
	};

	return (
		<>
			<Button block danger icon={<DeleteOutlined />} onClick={toggle}>
				删除历史记录
			</Button>

			<Drawer
				open={open}
				title="删除历史记录"
				width="100%"
				closable={false}
				classNames={{
					body: "py-16!",
					footer: "flex justify-end",
				}}
				footer={
					<Space>
						<Button disabled={deleting} onClick={toggle}>
							取消
						</Button>

						<Button type="primary" loading={deleting} onClick={onSubmit}>
							确定
						</Button>
					</Space>
				}
			>
				<Form
					form={form}
					initialValues={{
						timeRange: rangeOptions[0].value,
						customRange: [dayjs().subtract(1, "hour"), dayjs()],
					}}
				>
					<Space>
						<Form.Item name="timeRange" label="时间范围">
							<EcoSelect options={rangeOptions} />
						</Form.Item>

						{timeRange < 0 && (
							<Form.Item name="customRange">
								<RangePicker
									showTime
									disabledDate={(current) => current > dayjs().endOf("day")}
								/>
							</Form.Item>
						)}
					</Space>

					<Form.Item name="deleteFavorite" valuePropName="checked">
						<Checkbox>删除收藏</Checkbox>
					</Form.Item>
				</Form>
			</Drawer>
		</>
	);
};

export default Delete;
