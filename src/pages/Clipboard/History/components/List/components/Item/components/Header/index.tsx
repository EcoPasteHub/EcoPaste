import Icon from "@/components/Icon";
import type { HistoryItem } from "@/types/database";
import { Flex, Popconfirm } from "antd";
import { autoConvertBytes } from "arcdash";
import clsx from "clsx";
import type { FC } from "react";

interface HeaderProps extends HistoryItem {
	copy: () => void;
	collect: () => void;
	deleteItem: () => void;
}

const Header: FC<HeaderProps> = (props) => {
	const {
		type,
		value,
		size,
		createTime,
		isCollected,
		copy,
		collect,
		deleteItem,
	} = props;

	const { t, i18n } = useTranslation();

	const [copied, { toggle }] = useBoolean();

	useEffect(() => {
		if (!copied) return;

		setTimeout(toggle, 3000);
	}, [copied]);

	const renderType = () => {
		switch (type) {
			case "text":
				if (isURL(value)) {
					return t("clipboard.label.link");
				}

				if (isEmail(value)) {
					return t("clipboard.label.email");
				}

				if (isColor(value)) {
					return t("clipboard.label.color");
				}

				return t("clipboard.label.plain_text");
			case "rich-text":
				return t("clipboard.label.rich_text");
			case "html":
				return t("clipboard.label.html");
			case "image":
				return t("clipboard.label.image");
			case "files":
				return t("clipboard.label.n_files", {
					replace: [JSON.parse(value).length],
				});
		}
	};

	const renderSize = () => {
		if (type === "files" || type === "image") {
			return autoConvertBytes(size);
		}

		return t("clipboard.label.n_chars", {
			replace: [size],
		});
	};

	const renderPixel = () => {
		if (type !== "image") return;

		const { width, height } = props;

		return (
			<span>
				{width}×{height}
			</span>
		);
	};

	const handleCopy = () => {
		copy();
		toggle();
	};

	return (
		<Flex justify="space-between" className="color-2">
			<Flex align="center" gap={6} className="text-12">
				<span>{renderType()}</span>
				<span>{renderSize()}</span>
				{renderPixel()}
				<span>{dayjs(createTime).locale(i18n.language).fromNow()}</span>
			</Flex>

			<Flex
				align="center"
				gap={6}
				className="text-14"
				onClick={(event) => event.stopPropagation()}
				onDoubleClick={(event) => event.stopPropagation()}
			>
				{copied ? (
					<Icon
						size={15}
						name="i-iconamoon:check-circle-1-fill"
						className="color-success"
					/>
				) : (
					<Icon hoverable name="i-iconamoon:copy" onMouseDown={handleCopy} />
				)}

				<Icon
					hoverable
					name={isCollected ? "i-iconamoon:star-fill" : "i-iconamoon:star"}
					className={clsx({ "text-gold!": isCollected })}
					onMouseDown={collect}
				/>

				<Popconfirm
					title={t("clipboard.hints.delete_confirm")}
					placement="left"
					rootClassName="max-w-300"
					onConfirm={deleteItem}
				>
					<Icon
						hoverable
						size={15}
						name="i-iconamoon:trash-simple"
						className="hover:text-danger!"
					/>
				</Popconfirm>
			</Flex>
		</Flex>
	);
};

export default memo(Header);
