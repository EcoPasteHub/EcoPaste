import dayjs from "dayjs";
import zhCN from "dayjs/locale/zh-cn";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);
dayjs.locale(zhCN);

export default dayjs;
