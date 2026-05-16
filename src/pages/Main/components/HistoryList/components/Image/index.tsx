import { startDrag } from "@crabnebula/tauri-plugin-drag";
import { exists } from "@tauri-apps/plugin-fs";
import type { DragEvent, FC } from "react";
import LocalImage from "@/components/LocalImage";
import type { DatabaseSchemaHistory } from "@/types/database";

const Image: FC<DatabaseSchemaHistory<"image">> = (props) => {
	const { value } = props;

	const handleDragStart = (event: DragEvent<HTMLButtonElement>) => {
		event.preventDefault();
		event.stopPropagation();

		void (async () => {
			if (!(await exists(value))) return;

			await startDrag({
				icon: value,
				item: [value],
				mode: "copy",
			});
		})();
	};

	return (
		<button
			className="w-fit cursor-grab border-none bg-transparent p-0 text-left active:cursor-grabbing"
			draggable
			onDragStart={handleDragStart}
			type="button"
		>
			<LocalImage className="max-h-21.5" draggable={false} src={value} />
		</button>
	);
};

export default Image;
