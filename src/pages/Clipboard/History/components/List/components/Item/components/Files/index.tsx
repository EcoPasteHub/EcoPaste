import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Files: FC<HistoryItem> = (props) => {
	const { value } = props;

	const paths: string[] = JSON.parse(value);

	return <div>{paths.join("\n")}</div>;
};

export default memo(Files);
