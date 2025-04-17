import { getKeySymbol } from "@/components/ProShortcut/keys";
import { Card, Flex, Tag } from "antd";
import { castArray, union } from "lodash-es";
import Masonry from "react-masonry-css";

const Preset = () => {
	const { t } = useTranslation();

	const list = [
		{
			label: "preference.shortcut.preset.search",
			value: "Command+F",
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
			label: "preference.shortcut.preset.favorite_or_unfavorite",
			value: "Command+D",
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
			label: "preference.shortcut.preset.open_preferences",
			value: "Command+Comma",
		},
		{
			label: "preference.shortcut.preset.hide_window",
			value: ["Escape", "Command+W"],
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
			className="-mt-8 flex gap-8"
			columnClassName="flex flex-col gap-8 bg-clip-padding"
		>
			{list.map((item) => {
				const { label, value } = item;

				return (
					<Card key={label} size="small">
						<div className="mb-16 break-all">{t(label)}</div>

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
