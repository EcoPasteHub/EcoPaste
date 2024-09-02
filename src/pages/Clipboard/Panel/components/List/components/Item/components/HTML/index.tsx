import type { ClipboardItem } from "@/types/database";
import type { FC } from "react";

const HTML: FC<Partial<ClipboardItem>> = (props) => {
	const { value = "" } = props;
	const containerRef = useRef<HTMLDivElement>(null);

	useMount(() => {
		if (!containerRef.current) return;

		const links = containerRef.current.querySelectorAll("a");

		for (const link of links) {
			link.removeAttribute("target");
		}
	});

	useEventListener(
		"click",
		(event) => {
			const { target, metaKey, ctrlKey } = event;

			const link = (target as HTMLElement).closest("a");

			if (!link || metaKey || ctrlKey) return;

			event.preventDefault();
			event.stopPropagation();
		},
		{
			target: containerRef,
		},
	);

	return (
		<div
			ref={containerRef}
			dangerouslySetInnerHTML={{ __html: value }}
			className="translate-z-0"
		/>
	);
};

export default memo(HTML);
