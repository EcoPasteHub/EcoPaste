import type { HistoryItem } from "@/types/database";
import { open } from "@tauri-apps/api/shell";
import { Typography } from "antd";
import type { FC } from "react";

const Text: FC<HistoryItem> = (props) => {
	const { content = "" } = props;

	const renderContent = () => {
		if (isURL(content) || isEmail(content)) {
			const url = isURL(content) ? content : `mailto:${content}`;

			return (
				<span
					className="cursor-pointer text-primary"
					onMouseDown={() => open(url)}
				>
					{content}
				</span>
			);
		}

		return content;
	};

	return (
		<Typography.Paragraph ellipsis={{ rows: 4 }}>
			{renderContent()}
		</Typography.Paragraph>
	);
};

export default Text;
