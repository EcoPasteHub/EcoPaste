import { HistoryContext } from "@/pages/Clipboard/History";
import type { InputRef } from "antd";
import { Flex, Input } from "antd";
import clsx from "clsx";
import { useSnapshot } from "valtio";

const Search = ({ classNames }: { classNames: string }) => {
	const { state } = useContext(HistoryContext);

	const [value, setValue] = useState("");

	const inputRef = useRef<InputRef>(null);

	const labelRef = useRef<HTMLLabelElement>(null);

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
		<Flex
			ref={labelRef}
			align="center"
			component={"label"}
			className={clsx(`mx-auto w-336 ${classNames}`)}
		>
			<Input
				ref={inputRef}
				allowClear
				value={value}
				prefix={<i className="i-lucide:search" />}
				size="small"
				placeholder="搜索"
				onChange={(event) => {
					setValue(event.target.value);
				}}
			/>
		</Flex>
	);
};

export default Search;
