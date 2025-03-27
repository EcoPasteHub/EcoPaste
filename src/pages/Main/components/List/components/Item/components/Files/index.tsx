import type { HistoryTablePayload } from "@/types/database";
import type { FC } from "react";
import File from "./components/File";

const Files: FC<HistoryTablePayload> = (props) => {
	const { value } = props;

	const paths: string[] = JSON.parse(value);

	return paths.map((path) => {
		return <File key={path} path={path} count={paths.length} />;
	});
};

export default memo(Files);
