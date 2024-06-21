import Icon from "@/components/Icon";
import { HistoryContext } from "@/pages/Clipboard/History";
import { Flex, Input } from "antd";
import clsx from "clsx";

const Search = () => {
	const { state } = useContext(HistoryContext);

	const labelRef = useRef<HTMLLabelElement>(null);

	const [value, setValue] = useState("");

	const isFocusWithin = useFocusWithin(labelRef);

	useDebounceEffect(
		() => {
			state.value = value;
		},
		[value],
		{
			wait: 500,
		},
	);

	const focused = useCreation(() => {
		return isFocusWithin || Boolean(value);
	}, [isFocusWithin, value]);

	return (
		<Flex
			ref={labelRef}
			align="center"
			component={"label"}
			className={clsx("w-18 overflow-hidden", {
				"w-unset": focused,
			})}
		>
			<Icon
				hoverable
				name="i-lucide:search"
				hidden={focused}
				className="min-w-18"
			/>

			<Input
				allowClear
				value={value}
				size="small"
				placeholder="搜索"
				className="w-150"
				onChange={(event) => {
					setValue(event.target.value);
				}}
			/>
		</Flex>
	);
};

export default Search;
