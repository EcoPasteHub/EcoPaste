import type { HistoryTablePayload } from "@/types/database";
import DOMPurify from "dompurify";
import type { FC, MouseEvent } from "react";

const HTML: FC<Partial<HistoryTablePayload>> = (props) => {
	const { value = "" } = props;

	const handleClick = (event: MouseEvent) => {
		const { target, metaKey, ctrlKey } = event;

		const link = (target as HTMLElement).closest("a");

		if (!link || metaKey || ctrlKey) return;

		event.preventDefault();
		event.stopPropagation();
	};

	return (
		<div
			className="translate-z-0"
			dangerouslySetInnerHTML={{
				__html: DOMPurify.sanitize(value, {
					FORBID_ATTR: ["target", "controls", "autoplay", "autoPlay"],
				}),
			}}
			onClick={handleClick}
		/>
	);
};

export default memo(HTML);
