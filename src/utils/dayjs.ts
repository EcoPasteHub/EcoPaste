import dayjs from "dayjs";
import zhCN from "dayjs/locale/zh-cn";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);
dayjs.locale(zhCN);

export { dayjs };
