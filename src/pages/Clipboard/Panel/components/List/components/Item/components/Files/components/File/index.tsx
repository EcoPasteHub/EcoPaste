import { exists, mkdir, writeFile } from "@tauri-apps/plugin-fs";
import { Flex } from "antd";
import type { FC } from "react";
import { type Metadata, icon, metadata } from "tauri-plugin-fs-pro-api";
import Image from "../../../Image";

interface FileProps {
	path: string;
	count: number;
}

interface State extends Partial<Metadata> {
	iconPath?: string;
}

const File: FC<FileProps> = (props) => {
	const { path, count } = props;

	const state = useReactive<State>({});

	useAsyncEffect(async () => {
		if (!path) return;

		Object.assign(state, await metadata(path, { omitSize: false }));

		if (!state.isExist) return;

		await mkdir(getSaveIconPath(), { recursive: true });

		const iconPath = joinPath(getSaveIconPath(), `${state.extname}.png`);

		const existed = await exists(iconPath);

		if (existed) {
			state.iconPath = iconPath;

			return;
		}

		const bytes = await icon(path, 256);

		await writeFile(iconPath, bytes);

		state.iconPath = iconPath;
	}, [path]);

	const renderContent = () => {
		if (count === 1) {
			if (isImage(path)) {
				return <Image value={path} />;
			}
		}

		const height = 100 / Math.min(count, 3);

		return (
			<Flex align="center" gap={4} style={{ height: `${height}%` }}>
				<Image value={state.iconPath} className="h-full" />

				<span className="truncate">{state.fullName}</span>
			</Flex>
		);
	};

	return renderContent();
};

export default File;
