import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { NodeIndexOutlined, ReloadOutlined } from "@ant-design/icons";
import { open } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { dataDir, join } from "@tauri-apps/api/path";
import { Button, Flex, Space, Tooltip, message } from "antd";
import { isString } from "antd/es/button";
import type { FC } from "react";
import type { State } from "../..";

const SavePath: FC<{ state: State }> = (props) => {
	const { state } = props;
	const { t } = useTranslation();

	const handleChange = async (isDefault = false) => {
		try {
			const nextDir = isDefault
				? await dataDir()
				: await open({ directory: true });

			if (!isString(nextDir) || getSaveDataDir().startsWith(nextDir)) return;

			state.spinning = true;

			const dirName = await moveData(getSaveDataDir(), nextDir);

			if (!dirName) return;

			globalStore.env.saveDataDir = await join(nextDir, dirName);

			state.spinning = false;

			emit(LISTEN_KEY.CHANGE_DATA_FILE);

			message.success(
				t("preference.data_backup.storage_path.hints.save_success"),
			);
		} catch (error: any) {
			state.spinning = false;

			message.error(error);
		}
	};

	return (
		<ProList header={t("preference.data_backup.storage_path.title")}>
			<ProListItem
				title={
					<Flex vertical align="flex-start" gap={2}>
						{t("preference.data_backup.storage_path.label.storage_path")}
						<span
							className="color-3 hover:color-primary cursor-pointer break-all text-12 transition"
							onMouseDown={() => {
								previewPath(getSaveDataDir());
							}}
						>
							{getSaveDataDir(false)}
						</span>
					</Flex>
				}
			>
				<Space.Compact>
					<Tooltip
						title={t("preference.data_backup.storage_path.hints.custom_path")}
					>
						<Button
							icon={<NodeIndexOutlined />}
							onClick={() => handleChange()}
						/>
					</Tooltip>

					<Tooltip
						title={t("preference.data_backup.storage_path.hints.default_path")}
					>
						<Button
							icon={<ReloadOutlined className="text-14!" />}
							onClick={() => handleChange(true)}
						/>
					</Tooltip>
				</Space.Compact>
			</ProListItem>
		</ProList>
	);
};

export default SavePath;
