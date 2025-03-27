import { sep } from "@tauri-apps/api/path";
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

			state.iconPath = await icon(path, 256);
		} catch {
			Object.assign(state, {
				fullName: path.split(sep()).pop(),
			});
		}
	}, [path]);

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
