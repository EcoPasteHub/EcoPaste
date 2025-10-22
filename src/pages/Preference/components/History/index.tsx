import ProList from "@/components/ProList";
import type { Interval } from "@/types/shared";
import Delete from "./components/Delete";
import Duration from "./components/Duration";
import MaxCount from "./components/MaxCount";

const History = () => {
  const { t } = useTranslation();
  const timerRef = useRef<Interval>();

  useImmediate(clipboardStore.history, async () => {
    const { duration, maxCount } = clipboardStore.history;

    clearInterval(timerRef.current);

    if (duration === 0 && maxCount === 0) return;

    const delay = 1000 * 60 * 30;

    timerRef.current = setInterval(async () => {
      const list = await selectHistory((qb) => {
        return qb.where("favorite", "=", false);
      });

      for (const [index, item] of list.entries()) {
        const { createTime } = item;
        const diffDays = dayjs().diff(createTime, "days");
        const isExpired = duration > 0 && diffDays >= duration;
        const isOverMaxCount = maxCount > 0 && index >= maxCount;

        if (!isExpired && !isOverMaxCount) continue;

        deleteHistory(item);
      }
    }, delay);
  });

  return (
    <ProList footer={<Delete />} header={t("preference.history.history.title")}>
      <Duration />

      <MaxCount />
    </ProList>
  );
};

export default History;
