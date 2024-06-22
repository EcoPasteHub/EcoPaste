import type { HistoryItem } from "@/types/database";
import { Flex, Typography } from "antd";
import type { FC } from "react";
import tinycolor from "tinycolor2";

const Text: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	const renderColor = (color: tinycolor.Instance) => {
		const alpha = color.getAlpha();

		return (
			<Flex align="center" gap="small">
				<div
					className="relative h-22 w-22 rounded-full"
					style={{ backgroundColor: color.setAlpha(alpha / 2).toRgbString() }}
				>
					<span
						className="absolute inset-2 rounded-full"
						style={{ backgroundColor: value }}
					/>
				</div>

				{value}
			</Flex>
		);
	};

	const renderContent = () => {
		const color = tinycolor(value);

		if (color.isValid()) {
			return renderColor(color);
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
