import copyAudio from "@/assets/audio/copy.mp3";
import type { HistoryItem, TablePayload } from "@/types/database";
import { appWindow } from "@tauri-apps/api/window";
import { isEqual } from "lodash-es";
import { FixedSizeList } from "react-window";
import {
	onSomethingUpdate,
	readFilesURIs,
	readHtml,
	readImageBase64,
	readRtf,
	readText,
	startListening,
} from "tauri-plugin-clipboard-api";
import { useSnapshot } from "valtio";

interface State {
	historyList: HistoryItem[];
	previousPayload?: TablePayload;
}

const ClipboardHistory = () => {
	const { wakeUpKey } = useSnapshot(clipboardStore);

	const audioRef = useRef<HTMLAudioElement>(null);

	const state = useReactive<State>({
		historyList: [],
	});

	useMount(async () => {
		frostedWindow();

		getHistoryList();

		startListening();

		onSomethingUpdate(async (updateTypes) => {
			if (clipboardStore.enableAudio) {
				audioRef.current?.play();
			}

			let payload: TablePayload = {};

			if (updateTypes.files) {
				payload = {
					type: "files",
					content: JSON.stringify(await readFilesURIs()),
				};
			} else if (updateTypes.image) {
				payload = {
					type: "image",
					content: `data:image/png;base64, ${await readImageBase64()}`,
				};
			} else if (updateTypes.html) {
				payload = {
					type: "html",
					content: await readHtml(),
				};
			} else if (updateTypes.rtf) {
				payload = {
					type: "rtf",
					content: await readRtf(),
				};
			} else if (updateTypes.text) {
				payload = {
					type: "text",
					content: await readText(),
				};
			}

			if (isEqual(payload, state.previousPayload)) return;

			await insertSQL("history", payload);

			state.previousPayload = payload;

			getHistoryList();
		});
	});

	useRegister(toggleWindowVisible, [wakeUpKey]);

	const getHistoryList = async () => {
		state.historyList = await selectSQL<HistoryItem[]>("history");
	};

	return (
		<div
			className="h-screen rounded-8 p-8"
			onMouseDown={() => appWindow.startDragging()}
		>
			{/* <Icon hoverable name="i-lucide:search" />
			<Icon hoverable name="i-ri:pushpin-2-line" /> */}
			<audio ref={audioRef} src={copyAudio} />

			<FixedSizeList
				width={344}
				height={584}
				itemData={state.historyList}
				itemKey={(index, data) => data[index].id!}
				itemCount={state.historyList.length}
				itemSize={100}
			>
				{(item) => {
					const { index, style } = item;
					const { type, content = "", createTime } = item.data[index];

					return (
						<div style={style} className="not-last-of-type:pb-10">
							<div
								className="h-full rounded-5 bg-white shadow"
								onMouseDown={(event) => {
									event.stopPropagation();
								}}
							>
								{type}
								{createTime}
								{type === "image" ? (
									<img src={content} className="h-60" />
								) : (
									<div dangerouslySetInnerHTML={{ __html: content }} />
								)}
							</div>
						</div>
					);
				}}
			</FixedSizeList>
		</div>
	);
};

export default ClipboardHistory;
