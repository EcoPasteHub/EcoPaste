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
			state.search = value;
		},
		[value],
		{
			wait: 500,
		},
	);

	const focused = useCreation(() => {
		return isFocusWithin || Boolean(value);
	}, [isFocusWithin, value]);

	// 和实际焦点有时候不符.
	useEffect(() => {
		if (focused) {
			// console.log("focus");
			//TODO 获取前一个界面的焦点,临时保存
			activate();
		} else {
			// console.log("lost focus");
			noactivate();
			//TODO 但原逻辑的PIN按钮有逻辑冲突.
			//TODO 还原前一个界面的焦点
		}
	}, [focused]);

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
