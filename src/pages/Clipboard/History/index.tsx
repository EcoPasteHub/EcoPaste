import type { HistoryItem, TablePayload } from "@/types/database";
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

interface State {
	historyList: HistoryItem[];
	previousPayload?: TablePayload;
}

const ClipboardWindow = () => {
	const state = useReactive<State>({
		historyList: [],
	});

	useMount(async () => {
		frostedWindow();

		getHistoryList();

		startListening();

		onSomethingUpdate(async (updateTypes) => {
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

	const getHistoryList = async () => {
		state.historyList = await selectSQL<HistoryItem[]>("history");
	};

	return (
		<div data-tauri-drag-region className="h-screen rounded-8 p-8">
			{/* <Icon hoverable name="i-lucide:search" />
			<Icon hoverable name="i-ri:pushpin-2-line" /> */}

			<FixedSizeList
				itemData={state.historyList}
				height={584}
				itemCount={state.historyList.length}
				itemSize={100}
				width={344}
			>
				{(item) => {
					const { index, style } = item;
					const { id, type, content = "" } = item.data[index];

					return (
						<div key={id} style={style} className="not-last-of-type:pb-10">
							<div className="h-full rounded-5 bg-primary-2">
								{type}
								{type === "image" ? (
									<img src={content} alt="aaa" className="h-60" />
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

export default ClipboardWindow;
