import type { HistoryItem } from "@/types/database";
import { Flex, Typography } from "antd";
import clsx from "clsx";
import type { CSSProperties, FC } from "react";

const Text: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	const renderColor = () => {
		const className = "absolute rounded-inherit";
		const style: CSSProperties = {
			background: value,
		};

		return (
			<Flex align="center" gap="small">
				<div className="relative h-22 min-w-22 rounded-full">
					<span
						style={style}
						className={clsx(className, "inset-0 opacity-50")}
					/>

					<span style={style} className={clsx(className, "inset-2")} />
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
