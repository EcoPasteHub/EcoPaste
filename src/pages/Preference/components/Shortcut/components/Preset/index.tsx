import { getKeySymbol } from "@/components/ProShortcut/keyboard";
import { Card, Flex, Tag } from "antd";
import { castArray, union } from "lodash-es";
import Masonry from "react-masonry-css";

const Preset = () => {
	const { t } = useTranslation();

	const list = [
		{
			label: "preference.shortcut.preset.copy",
			value: PRESET_SHORTCUT.COPY,
		},
		{
			label: "preference.shortcut.preset.search",
			value: PRESET_SHORTCUT.SEARCH,
		},
		{
			label: "preference.shortcut.preset.select_item",
			value: ["tab", "shift.tab"],
		},
		{
			label: "preference.shortcut.preset.select_group",
			value: ["uparrow", "downarrow"],
		},
		{
			label: "preference.shortcut.preset.paste",
			value: "enter",
		},
		{
			label: "preference.shortcut.preset.delete",
			value: ["delete", "backspace"],
		},
		{
			label: "preference.shortcut.preset.favorite",
			value: PRESET_SHORTCUT.FAVORITE,
		},
		{
			label: "preference.shortcut.preset.preview_image",
			value: "space",
		},
		{
			label: "preference.shortcut.preset.back_to_top",
			value: "Home",
		},
		{
			label: "preference.shortcut.preset.fixed_window",
			value: PRESET_SHORTCUT.FIXED_WINDOW,
		},
		{
			label: "preference.shortcut.preset.open_preferences",
			value: PRESET_SHORTCUT.OPEN_PREFERENCES,
		},
		{
			label: "preference.shortcut.preset.hide_window",
			value: ["esc", PRESET_SHORTCUT.HIDE_WINDOW],
		},
	].map(({ label, value }) => ({
		label,
		value: union(
			castArray(value).map((item) => {
				return item.split(".").map(getKeySymbol).join(" + ");
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
