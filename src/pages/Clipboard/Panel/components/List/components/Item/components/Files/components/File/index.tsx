import { mkdir, writeFile } from "@tauri-apps/plugin-fs";
import { Flex } from "antd";
import clsx from "clsx";
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
		try {
			const data = await metadata(path, { omitSize: true });

			Object.assign(state, data);

			if (isLinux()) return;

			state.iconPath = await getIconPath();
		} catch {
			Object.assign(state, {
				fullName: path.split("/").pop(),
			});
		}
	}, [path]);

	const getIconPath = async () => {
		const iconPath = joinPath(getSaveIconPath(), `${getIconName()}.png`);

		await mkdir(getSaveIconPath(), { recursive: true });

		const bytes = await icon(path, 256);

		await writeFile(iconPath, bytes);

		return iconPath;
	};

	const getIconName = () => {
		const { isDir, extname, name, fullName } = state;

		const isMacApp = isMac() && extname === "app";
		const isWinApp = isWin() && extname === "exe";

		if (isMacApp || isWinApp) {
			return fullName;
		}

		if (isDir) {
			return "__ECOPASTE_DIRECTORY__";
		}

		if (!extname) {
			return name;
		}

		return extname;
	};

	const renderContent = () => {
		const height = 100 / Math.min(count, 3);

		if (state.isExist && count === 1) {
			if (isImage(path)) {
				return <Image value={path} />;
			}
		}

		return (
			<Flex align="center" gap={4} style={{ height: `${height}%` }}>
				{state.isExist && <Image value={state.iconPath} className="h-full" />}

				<span
					className={clsx("truncate", {
						"text-danger line-through": !state.isExist,
					})}
				>
					{state.fullName}
				</span>
			</Flex>
		);
	};

	return renderContent();
};

export default File;
