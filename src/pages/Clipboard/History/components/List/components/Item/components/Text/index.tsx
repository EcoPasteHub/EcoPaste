import type { HistoryItem } from "@/types/database";
import { Flex, Typography } from "antd";
import type { CSSProperties, FC } from "react";

const Text: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	const renderColor = () => {
		const style: CSSProperties = {
			backgroundColor: value,
			backgroundImage: value,
		};

		return (
			<Flex align="center" gap="small">
				<div className="relative h-22 min-w-22 overflow-hidden rounded-full">
					<span className="absolute inset-0 opacity-50" style={style} />

					<span className="absolute inset-2 rounded-full" style={style} />
				</div>

				{value}
			</Flex>
		);
	};

	const renderContent = () => {
		if (isColor(value)) {
			return renderColor();
		}

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

export default memo(Text);
