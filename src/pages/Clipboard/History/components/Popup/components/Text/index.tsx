import type { HistoryItem } from "@/types/database";
import { Typography } from "antd";
import type { FC } from "react";

const Text: FC<HistoryItem> = (props) => {
	const { content = "" } = props;

	const renderContent = () => {
		if (isURL(content) || isEmail(content)) {
			const url = isURL(content) ? content : `mailto:${content}`;

			return (
				<a href={url} target="_blank" rel="noreferrer">
					{content}
				</a>
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
