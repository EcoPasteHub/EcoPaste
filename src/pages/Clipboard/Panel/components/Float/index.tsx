import Icon from "@/components/Icon";
import { Flex } from "antd";
import clsx from "clsx";
import { useSnapshot } from "valtio";
import Group from "../Group";
import List from "../List";
import Pin from "../Pin";
import Search from "../Search";

const Float = () => {
	const { search } = useSnapshot(clipboardStore);

	return (
		<div className={clsx("h-screen bg-1 p-3", { "rounded-10": !isWin() })}>
			<Flex
				data-tauri-drag-region
				vertical
				gap={12}
				className={clsx("h-full py-9", {
					"flex-col-reverse": search.position === "bottom",
				})}
			>
				<Search className="mx-9" />

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
						className="px-9"
					>
						<Group />

						<Flex align="center" gap={4} className="color-2 text-18">
							<Pin />

							<Icon
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
