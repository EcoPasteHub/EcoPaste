import Icon from "@/components/Icon";
import type { InputRef } from "antd";
import { Input } from "antd";
import type { FC, HTMLAttributes } from "react";
import { ClipboardPanelContext } from "../..";

const Search: FC<HTMLAttributes<HTMLDivElement>> = (props) => {
	const { state } = useContext(ClipboardPanelContext);
	const inputRef = useRef<InputRef>(null);
	const [value, setValue] = useState("");
	const { t } = useTranslation();

	useFocus({
		onFocus() {
			if (clipboardStore.search.defaultFocus) {
				inputRef.current?.focus();
			}
		},
		onBlur() {
			inputRef.current?.blur();

			if (clipboardStore.search.autoClear) {
				setValue("");
			}
		},
	});

	useDebounceEffect(
		() => {
			state.search = value;
		},
		[value],
		{ wait: 500 },
	);

	useOSKeyPress(["meta.f", "ctrl.f"], () => {
		inputRef.current?.focus();
	});

	return (
		<div {...props}>
			<Input
				ref={inputRef}
				allowClear
				value={value}
				prefix={<Icon name="i-lucide:search" />}
				size="small"
				placeholder={t("clipboard.hints.search_placeholder")}
				onChange={(event) => {
					setValue(event.target.value);
				}}
			/>
		</div>
	);
};

export default Search;
