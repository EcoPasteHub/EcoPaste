import { startDrag } from "@crabnebula/tauri-plugin-drag";
import { exists } from "@tauri-apps/plugin-fs";
import type { DragEvent, FC } from "react";
import { getDefaultSaveImagePath } from "tauri-plugin-clipboard-x-api";
import LocalImage from "@/components/LocalImage";
import type { DatabaseSchemaHistory } from "@/types/database";
import { join } from "@/utils/path";

const isAbsolutePath = (path: string) => {
	if (/^[a-zA-Z]:[\\/]/.test(path)) return true;
	if (path.startsWith("\\\\")) return true;
	if (path.startsWith("/")) return true;
	return false;
};

const Image: FC<DatabaseSchemaHistory<"image">> = (props) => {
	const { value } = props;

	const handleDragStart = (event: DragEvent<HTMLButtonElement>) => {
		event.preventDefault();
		event.stopPropagation();

		void (async () => {
			let dragPath = value;

			try {
				const saveImagePath = await getDefaultSaveImagePath();
				if (
					saveImagePath &&
					!isAbsolutePath(value) &&
					!value.startsWith(saveImagePath)
				) {
					dragPath = join(saveImagePath, value);
				}
			} catch {}

			if (!(await exists(dragPath))) return;

			await startDrag({
				icon: dragPath,
				item: [dragPath],
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
