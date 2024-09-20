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

	const { t } = useTranslation();

	useEffect(form.resetFields, [open]);

	const rangeOptions: DefaultOptionType[] = [
		{
			label: t("preference.history.history.label.time_range_opt.last_hour"),
			value: 1,
		},
		{
			label: t("preference.history.history.label.time_range_opt.last_24_hours"),
			value: 24,
		},
		{
			label: t("preference.history.history.label.time_range_opt.last_7_days"),
			value: 7 * 24,
		},
		{
			label: t("preference.history.history.label.time_range_opt.last_30_days"),
			value: 30 * 24,
		},
		{
			label: t("preference.history.history.label.time_range_opt.unlimited"),
			value: 0,
		},
		{
			label: t("preference.history.history.label.time_range_opt.custom"),
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
		message.success(t("preference.history.history.hints.delete_success"));
		emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);
	};

	return (
		<>
			<Button block danger icon={<DeleteOutlined />} onClick={toggle}>
				{t("preference.history.history.button.goto_delete")}
			</Button>

			<Drawer
				open={open}
				title={t("preference.history.history.label.delete_title")}
				width="100%"
				closable={false}
				classNames={{
					body: "py-16!",
					footer: "flex justify-end",
				}}
				footer={
					<Space>
						<Button disabled={deleting} onClick={toggle}>
							{t("preference.history.history.button.cancel_delete")}
						</Button>

						<Button type="primary" loading={deleting} onClick={onSubmit}>
							{t("preference.history.history.button.confirm_delete")}
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
						<Form.Item
							name="timeRange"
							label={t("preference.history.history.label.time_range")}
						>
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
						<Checkbox>
							{t("preference.history.history.label.include_favorite")}
						</Checkbox>
					</Form.Item>
				</Form>
			</Drawer>
		</>
	);
};

export default Delete;
