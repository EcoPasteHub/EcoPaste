import UnoIcon from "@/components/UnoIcon";
import { Flex } from "antd";
import clsx from "clsx";
import { useSnapshot } from "valtio";
import GroupList from "../GroupList";
import HistoryList from "../HistoryList";
import SearchInput from "../SearchInput";
import WindowPin from "../WindowPin";

const StandardMode = () => {
	const { search } = useSnapshot(clipboardStore);

	return (
		<Flex
			data-tauri-drag-region
			vertical
			gap={12}
			className={clsx("h-screen bg-color-1 py-3", {
				"rounded-2.5": !isWin,
				"b b-color-1": isLinux,
				"flex-col-reverse": search.position === "bottom",
			})}
		>
			<SearchInput className="mx-3" />

			<Flex
				data-tauri-drag-region
				vertical
				gap={12}
				className="flex-1 overflow-hidden"
			>
				<Flex
					data-tauri-drag-region
					align="center"
					justify="space-between"
					gap="small"
					className="overflow-hidden px-3"
				>
					<GroupList />

					<Flex align="center" gap={4} className="text-color-2 text-lg">
						<WindowPin />

						<UnoIcon
							hoverable
							name="i-lets-icons:setting-alt-line"
							onClick={() => {
								showWindow("preference");
							}}
						/>
					</Flex>
				</Flex>

				<HistoryList />
			</Flex>
		</Flex>
	);
};

export default StandardMode;
