import Icon from "@/components/Icon";
import ProList from "@/components/ProList";
import type { Store } from "@/types/store";
import { emit } from "@tauri-apps/api/event";
import { readTextFile } from "@tauri-apps/api/fs";
import { Flex, List, message } from "antd";
import { merge } from "lodash-es";
import type { FC } from "react";
import type { State } from "../..";

const Manual: FC<{ state: State }> = (props) => {
	const { state } = props;

	const handleImport = async () => {
		state.spinning = true;

		const result = await importData();

		state.spinning = false;

		if (!result) return;

		const content = await readTextFile(getBackupStorePath());

		const store = JSON.parse(content) as Store;

		merge(globalStore, store.globalStore);
		merge(clipboardStore, store.clipboardStore);

		emit(LISTEN_KEY.CHANGE_DATA);

		message.success("导入成功");
	};

	const handleExport = async () => {
		state.spinning = true;

		await exportData();

		state.spinning = false;
	};

	const mapList = [
		{
			label: "导入数据",
			icon: "i-hugeicons:database-import",
			event: handleImport,
		},
		{
			label: "导出数据",
			icon: "i-hugeicons:database-export",
			event: handleExport,
		},
	];

	return (
		<ProList header="导入和导出">
			<List.Item className="p-16!">
				<Flex gap="middle" className="w-full">
					{mapList.map((item) => {
						const { label, icon, event } = item;

						return (
							<Flex
								key={label}
								vertical
								align="center"
								justify="center"
								gap="small"
								className="b b-color-2 hover:b-primary h-102 flex-1 cursor-pointer rounded-8 bg-3 px-8 text-center transition hover:text-primary"
								onClick={event}
							>
								<Icon name={icon} size={26} />
								{label}
							</Flex>
						);
					})}
				</Flex>
			</List.Item>
		</ProList>
	);
};

export default Manual;
