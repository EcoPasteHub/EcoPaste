import type { HistoryItem } from "@/types/database";
import type { FC } from "react";
import { EMFJS, RTFJS, WMFJS } from "rtf.js";
import HTML from "../HTML";

RTFJS.loggingEnabled(false);
WMFJS.loggingEnabled(false);
EMFJS.loggingEnabled(false);

const stringToArrayBuffer = (rtf: string) => {
	const buffer = new ArrayBuffer(rtf.length);

	const bufferView = new Uint8Array(buffer);

	for (let i = 0; i < rtf.length; i++) {
		bufferView[i] = rtf.charCodeAt(i);
	}

	return buffer;
};

const RichText: FC<HistoryItem> = (props) => {
	const { value } = props;

	const [parsedHTML, setParsedHTML] = useState("");

	useMount(async () => {
		const doc = new RTFJS.Document(stringToArrayBuffer(value), {});

		const elements = await doc.render();

		const parsedHTML = elements
			.map(({ outerHTML }) => outerHTML)
			.join("")
			.replace(/(\d+)pt/g, "$1px");

		setParsedHTML(parsedHTML);
	});

	return <HTML value={parsedHTML} />;
};

export default memo(RichText);
