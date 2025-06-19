import UnoIcon from "@/components/UnoIcon";
import { Flex } from "antd";
import clsx from "clsx";
import { useSnapshot } from "valtio";
import Group from "../Group";
import List from "../List";
import Pin from "../Pin";
import Search from "../Search";

const Float = () => {
	const { search } = useSnapshot(clipboardStore);
	const { appearance } = useSnapshot(globalStore);

	return (
		<div
			className={clsx("h-screen bg-color-2", {
				"rounded-2.5": !isWin,
				"b b-color-1": isLinux,
				"b b-color-2": appearance.isDark && !isLinux,
			})}
		>
			<Flex
				data-tauri-drag-region
				vertical
				gap={12}
				className={clsx("h-full py-3", {
					"flex-col-reverse": search.position === "bottom",
				})}
			>
				<Search className="mx-3" />

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
						className="px-3"
					>
						<Group />

						<Flex align="center" gap={4} className="text-color-2 text-lg">
							<Pin />

							<UnoIcon
								hoverable
								name="i-lets-icons:setting-alt-line"
								onClick={() => {
									showWindow("preference");
								}}
							/>
						</Flex>
					</Flex>

					<List />
				</Flex>
			</Flex>
		</div>
	);
};

export default Float;
