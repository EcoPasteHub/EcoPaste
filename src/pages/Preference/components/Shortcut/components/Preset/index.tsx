import { getKeySymbol } from "@/components/ProShortcut/keys";
import { Card, Flex, Tag } from "antd";
import { castArray, union } from "lodash-es";
import Masonry from "react-masonry-css";

const Preset = () => {
	const { t } = useTranslation();

	const list = [
		{
			label: "preference.shortcut.preset.search",
			value: isMac ? "Command+F" : "Ctrl+F",
		},
		{
			label: "preference.shortcut.preset.select_item",
			value: ["Tab", "Shift+Tab"],
		},
		{
			label: "preference.shortcut.preset.select_group",
			value: ["ArrowUp", "ArrowDown"],
		},
		{
			label: "preference.shortcut.preset.paste",
			value: "Enter",
		},
		{
			label: "preference.shortcut.preset.delete",
			value: ["Delete", "Backspace"],
		},
		{
			label: "preference.shortcut.preset.favorite",
			value: isMac ? "Command+D" : "Ctrl+D",
		},
		{
			label: "preference.shortcut.preset.preview_image",
			value: "Space",
		},
		{
			label: "preference.shortcut.preset.back_to_top",
			value: "Home",
		},
		{
			label: "preference.shortcut.preset.fixed_window",
			value: isMac ? "Command+P" : "Ctrl+P",
		},
		{
			label: "preference.shortcut.preset.open_preferences",
			value: isMac ? "Command+Comma" : "Ctrl+Comma",
		},
		{
			label: "preference.shortcut.preset.hide_window",
			value: ["Escape", isMac ? "Command+W" : "Ctrl+W"],
		},
	].map(({ label, value }) => ({
		label,
		value: union(
			castArray(value).map((item) => {
				return item.split("+").map(getKeySymbol).join(" + ");
			}),
		),
	}));

	return (
		<Masonry
			breakpointCols={3}
			className="-mt-2 flex gap-2"
			columnClassName="flex flex-col gap-2 bg-clip-padding"
		>
			{list.map((item) => {
				const { label, value } = item;

				return (
					<Card key={label} size="small">
						<div className="mb-4 break-all">{t(label)}</div>

						<Flex wrap gap="small">
							{value.map((item) => (
								<Tag key={item} className="m-0">
									{item}
								</Tag>
							))}
						</Flex>
					</Card>
				);
			})}
		</Masonry>
	);
};

export default Preset;
