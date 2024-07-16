import Icon from "@/components/Icon";
import { HistoryContext } from "@/pages/Clipboard/History";
import type { InputProps, InputRef } from "antd";
import { Input } from "antd";
import type { FC } from "react";
import { useSnapshot } from "valtio";

const Search: FC<InputProps> = (props) => {
	const { state } = useContext(HistoryContext);

	const [value, setValue] = useState("");

	const inputRef = useRef<InputRef>(null);

	const { isFocus, defaultFocus } = useSnapshot(clipboardStore);

	useEffect(() => {
		if (isFocus && defaultFocus === "search") {
			inputRef.current?.focus();
		}
	}, [isFocus, defaultFocus]);

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
