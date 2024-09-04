import Icon from "@/components/Icon";
import type { InputRef } from "antd";
import { Input } from "antd";
import type { FC, HTMLAttributes } from "react";
import { ClipboardPanelContext } from "../..";

const Search: FC<HTMLAttributes<HTMLDivElement>> = (props) => {
	const { state } = useContext(ClipboardPanelContext);
	const inputRef = useRef<InputRef>(null);
	const [value, setValue] = useState<string>();
	const [isComposition, { setTrue, setFalse }] = useBoolean();
	const { t } = useTranslation();

	useFocus({
		onFocus() {
			if (clipboardStore.search.defaultFocus) {
				inputRef.current?.focus();
			} else {
				inputRef.current?.blur();
			}
		},
		onBlur() {
			if (clipboardStore.search.autoClear) {
				setValue(undefined);
			}
		},
	});

	useOSKeyPress(["meta.f", "ctrl.f"], () => {
		inputRef.current?.focus();
	});

	useEffect(() => {
		if (isComposition) return;

		state.search = value;
	}, [value, isComposition]);

	return (
		<div {...props}>
			<Input
				ref={inputRef}
				allowClear
				value={value}
				prefix={<Icon name="i-lucide:search" />}
				size="small"
				placeholder={t("clipboard.hints.search_placeholder")}
				onCompositionStart={setTrue}
				onCompositionEnd={setFalse}
				onChange={(event) => {
					setValue(event.target.value);
				}}
			/>
		</div>
	);
};

export default Search;
