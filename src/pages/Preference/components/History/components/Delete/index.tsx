import { DeleteOutlined } from "@ant-design/icons";
import { emit } from "@tauri-apps/api/event";
import { useBoolean } from "ahooks";
import {
  Button,
  Checkbox,
  DatePicker,
  Drawer,
  Form,
  message,
  Space,
} from "antd";
import type { DefaultOptionType } from "antd/es/select";
import type { Dayjs } from "dayjs";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import AdaptiveSelect from "@/components/AdaptiveSelect";
import { LISTEN_KEY } from "@/constants";
import { deleteHistory, selectHistory } from "@/database/history";
import { dayjs, formatDate } from "@/utils/dayjs";

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
    try {
      const { timeRange, customRange, deleteFavorite } = form.getFieldsValue();

      setTrue();

      let range: Dayjs[] = [];

      if (timeRange < 0) {
        range = customRange;
      } else {
        range = [dayjs().subtract(timeRange, "hour"), dayjs()];
      }

      const formatRange = range.map((item) => formatDate(item));

      const list = await selectHistory();

      for await (const item of list) {
        const { favorite, createTime } = item;

        if (favorite && !deleteFavorite) continue;

        const isBetween = dayjs(createTime).isBetween(
          formatRange[0],
          formatRange[1],
          null,
          "[]",
        );

        if (timeRange === 0 || isBetween) {
          deleteHistory(item);
        }
      }

      toggle();
      message.success(t("preference.history.history.hints.delete_success"));
      emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);
    } catch (error) {
      message.error(String(error));
    } finally {
      setFalse();
    }
  };

  return (
    <>
      <Button block danger icon={<DeleteOutlined />} onClick={toggle}>
        {t("preference.history.history.button.goto_delete")}
      </Button>

      <Drawer
        classNames={{
          body: "py-4!",
          footer: "flex justify-end",
        }}
        closable={false}
        footer={
          <Space>
            <Button disabled={deleting} onClick={toggle}>
              {t("preference.history.history.button.cancel_delete")}
            </Button>

            <Button loading={deleting} onClick={onSubmit} type="primary">
              {t("preference.history.history.button.confirm_delete")}
            </Button>
          </Space>
        }
        open={open}
        title={t("preference.history.history.label.delete_title")}
        width="100%"
      >
        <Form
          form={form}
          initialValues={{
            customRange: [dayjs().subtract(1, "hour"), dayjs()],
            timeRange: rangeOptions[0].value,
          }}
        >
          <Space>
            <Form.Item
              label={t("preference.history.history.label.time_range")}
              name="timeRange"
            >
              <AdaptiveSelect options={rangeOptions} />
            </Form.Item>

            {timeRange < 0 && (
              <Form.Item name="customRange">
                <RangePicker
                  disabledDate={(current) => current > dayjs().endOf("day")}
                  showTime
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
