import Icon from "@/components/Icon";
import { HistoryContext } from "@/pages/Clipboard/History";
import type { InputProps, InputRef } from "antd";
import { Input } from "antd";
import { isNil } from "lodash-es";
import type { FC } from "react";

const Search: FC<InputProps> = (props) => {
	const { state } = useContext(HistoryContext);

	const inputRef = useRef<InputRef>(null);

	const [value, setValue] = useState("");

	useFocus({
		onFocus() {
			const { defaultFocus, activeIndex } = clipboardStore;

			if (defaultFocus === "search") {
				inputRef.current?.focus();
			} else if (isNil(activeIndex)) {
				clipboardStore.activeIndex = 0;
			}
		},
		onBlur() {
			inputRef.current?.blur();

			clipboardStore.activeIndex = -1;
		},
	});

	useDebounceEffect(
		() => {
			state.search = value;
		},
		[value],
		{
			wait: 500,
		},
	);

	return (
		<Input
			{...props}
			ref={inputRef}
			allowClear
			value={value}
			prefix={<Icon name="i-lucide:search" />}
			size="small"
			placeholder="搜索"
			className="m-auto w-336"
			onChange={(event) => {
				setValue(event.target.value);
			}}
		/>
	);
};

export default Search;
