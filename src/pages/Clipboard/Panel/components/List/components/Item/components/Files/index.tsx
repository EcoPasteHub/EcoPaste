import type { ClipboardItem } from "@/types/database";
import type { FC } from "react";
import Image from "../Image";

const Files: FC<ClipboardItem> = (props) => {
	const { value } = props;

	const paths: string[] = JSON.parse(value);

	const renderContent = () => {
		if (paths.length === 1) {
			const [path] = paths;

			if (isImage(path)) {
				return <Image value={path} />;
			}

			return path;
		}

		return paths.join("\n");
	};

	return renderContent();
};

export default memo(Files);
