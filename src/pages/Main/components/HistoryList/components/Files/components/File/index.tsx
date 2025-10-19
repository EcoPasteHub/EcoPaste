import LocalImage from "@/components/LocalImage";
import { Flex } from "antd";
import clsx from "clsx";
import type { FC } from "react";
import {
	type Metadata,
	fullName,
	icon,
	metadata,
} from "tauri-plugin-fs-pro-api";

interface FileProps {
	path: string;
	count: number;
}

interface State extends Partial<Metadata> {
	icon?: string;
}

const File: FC<FileProps> = (props) => {
	const { path, count } = props;

	const state = useReactive<State>({});

	useAsyncEffect(async () => {
		try {
			const data = await metadata(path, { omitSize: true });

			Object.assign(state, data);

			if (isLinux) return;

			state.icon = await icon(path, { size: 256 });
		} catch {
			state.fullName = await fullName(path);
		}
	}, [path]);

	const renderContent = () => {
		if (state.isExist && count === 1 && isImage(path)) {
			return <LocalImage src={path} className="max-h-21.5" />;
		}

		return (
			<div className="flex-1 overflow-hidden">
				<Flex align="center" gap={4} className="h-full">
					{state.icon && <LocalImage src={state.icon} className="h-full" />}

					<span
						className={clsx("truncate", {
							"text-danger line-through": !state.isExist,
						})}
					>
						{state.fullName}
					</span>
				</Flex>
			</div>
		);
	};

	return renderContent();
};

export default File;
