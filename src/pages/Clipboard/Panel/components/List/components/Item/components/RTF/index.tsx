import type { ClipboardItem } from "@/types/database";
import type { FC } from "react";
import { EMFJS, RTFJS, WMFJS } from "rtf.js";
import HTML from "../HTML";

RTFJS.loggingEnabled(false);
WMFJS.loggingEnabled(false);
EMFJS.loggingEnabled(false);

const RTF: FC<ClipboardItem> = (props) => {
	const { value } = props;

	const [parsedHTML, setParsedHTML] = useState("");

	useMount(async () => {
		const doc = new RTFJS.Document(stringToArrayBuffer(value), {});

		const elements = await doc.render();

		pt2px(elements);

		const parsedHTML = elements.map(({ outerHTML }) => outerHTML).join("");

		setParsedHTML(parsedHTML);
	});

	const pt2px = (elements: Element[]) => {
		for (const element of elements) {
			let style = element.getAttribute("style");

			style = style?.replace(/(\d+)pt/g, "px") ?? "";

			element.setAttribute("style", style);

			pt2px([...element.children]);
		}
	};

	return <HTML value={parsedHTML} />;
};

export default memo(RTF);
