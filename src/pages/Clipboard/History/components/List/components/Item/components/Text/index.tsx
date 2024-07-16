import type { HistoryItem } from "@/types/database";
import { Flex, Typography } from "antd";
import clsx from "clsx";
import type { CSSProperties, FC } from "react";

const Text: FC<HistoryItem> = (props) => {
	const { value } = props;

	const renderColor = () => {
		const className = "absolute rounded-full";
		const style: CSSProperties = {
			background: value,
		};

		return (
			<Flex align="center" gap="small">
				<div className="relative h-22 min-w-22">
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

		return value;
	};

	return (
		<Typography.Paragraph ellipsis={{ rows: 4 }} className="leading-unset">
			{renderContent()}
		</Typography.Paragraph>
	);
};

export default memo(Text);
