import { startDrag } from "@crabnebula/tauri-plugin-drag";
import { exists } from "@tauri-apps/plugin-fs";
import type { DragEvent, FC } from "react";
import { fullName } from "tauri-plugin-fs-pro-api";
import type { DatabaseSchemaHistory } from "@/types/database";
import { isImage } from "@/utils/is";
import File from "./components/File";

const Files: FC<DatabaseSchemaHistory<"files">> = (props) => {
	const { value } = props;

	const getClassName = () => {
		if (value.length === 1) {
			if (isImage(value[0])) {
				return "max-h-21.5";
			}

			return "h-7";
		}

		if (value.length === 2) {
			return "h-14";
		}

		return "h-21.5";
	};

	const handleDragStart = (event: DragEvent<HTMLButtonElement>) => {
		event.preventDefault();
		event.stopPropagation();

		void (async () => {
			const dragPaths = (
				await Promise.all(
					value.map(async (path) => {
						try {
							const resolvedPath = await fullName(path);

							if (await exists(resolvedPath)) {
								return resolvedPath;
							}
						} catch {}

						if (await exists(path)) {
							return path;
						}

						return null;
					}),
				)
			).filter((path): path is string => Boolean(path));

			if (!dragPaths.length) return;

			await startDrag({
				icon: dragPaths[0],
				item: dragPaths,
				mode: "copy",
			});
		})();
	};

	return (
		<button
			aria-label="拖动导出文件"
			className={`${getClassName()} w-full cursor-grab border-0 bg-transparent p-0 text-left active:cursor-grabbing`}
			draggable
			onDragStart={handleDragStart}
			type="button"
		>
			{value.map((path) => {
				return <File count={value.length} key={path} path={path} />;
			})}
		</button>
	);
};

export default Files;
