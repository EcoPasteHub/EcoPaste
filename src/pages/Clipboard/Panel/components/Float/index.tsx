import Icon from "@/components/Icon";
import {
	PhysicalPosition,
	appWindow,
	availableMonitors,
} from "@tauri-apps/api/window";
import { Flex } from "antd";
import clsx from "clsx";
import { useSnapshot } from "valtio";
import { ClipboardPanelContext } from "../..";
import Group from "../Group";
import List from "../List";
import Pin from "../Pin";
import Search from "../Search";

const Float = () => {
	const { state } = useContext(ClipboardPanelContext);
	const { shortcut } = useSnapshot(globalStore);
	const { search } = useSnapshot(clipboardStore);

	useRegister(async () => {
		const focused = await appWindow.isFocused();

		if (!focused) {
			const { window } = clipboardStore;

			if (window.position !== "remember") {
				const monitors = await availableMonitors();

				if (!monitors.length) return;

				const { width, height } = await appWindow.innerSize();

				const [x, y] = await getMouseCoords();

				for (const monitor of monitors) {
					const {
						scaleFactor,
						position: { x: posX, y: posY },
						size: { width: screenWidth, height: screenHeight },
					} = monitor;

					const factor = isMac() ? scaleFactor : 1;
					let coordX = x * factor;
					let coordY = y * factor;

					if (
						coordX < posX ||
						coordY < posY ||
						coordX > posX + screenWidth ||
						coordY > posY + screenHeight
					) {
						continue;
					}

					if (window.position === "follow") {
						coordX = Math.min(coordX, posX + screenWidth - width);
						coordY = Math.min(coordY, posY + screenHeight - height);
					} else if (window.position === "center") {
						coordX = posX + (screenWidth - width) / 2;
						coordY = posY + (screenHeight - height) / 2;
					}

					appWindow.setPosition(new PhysicalPosition(coordX, coordY));

					break;
				}
			}
		}

		toggleWindowVisible();
	}, [shortcut.clipboard]);

	useFocus({
		onBlur() {
			if (state.pin) return;

			hideWindow();
		},
	});

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
