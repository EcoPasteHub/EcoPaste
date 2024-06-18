import type { HistoryItem } from "@/types/database";
import { Typography } from "antd";
import type { FC } from "react";

const Text: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	const renderContent = () => {
		if (isURL(value) || isEmail(value)) {
			const url = isURL(value) ? value : `mailto:${value}`;

			return (
				<a href={url} target="_blank" rel="noreferrer">
					{value}
				</a>
			);
		}

		return value;
	};

	return (
		<Typography.Paragraph ellipsis={{ rows: 4 }}>
			{renderContent()}
		</Typography.Paragraph>
	);
};

export default Text;
