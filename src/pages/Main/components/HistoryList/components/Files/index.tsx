import type { DatabaseSchemaHistory } from "@/types/database";
import { Flex } from "antd";
import type { FC } from "react";
import File from "./components/File";

const Files: FC<DatabaseSchemaHistory<"files">> = (props) => {
	const { value } = props;

	const getClassName = () => {
		if (value.length === 1) {
			if (isImage(value[0])) {
				return "max-h-21.5";
			}

			return "h-7";
		}

		if (value.length === 2) {
			return "h-14";
		}

		return "h-21.5";
	};

	return (
		<Flex vertical align="start" gap={4} className={getClassName()}>
			{value.map((path) => {
				return <File key={path} path={path} count={value.length} />;
			})}
		</Flex>
	);
};

export default Files;
