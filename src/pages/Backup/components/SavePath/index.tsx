import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { NodeIndexOutlined, ReloadOutlined } from "@ant-design/icons";
import { open } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { dataDir as tauriDataDir } from "@tauri-apps/api/path";
import { Button, Space, Tooltip, message } from "antd";
import { isString } from "antd/es/button";
import { isEqual } from "lodash-es";
import type { FC } from "react";
import type { State } from "../..";

const SavePath: FC<{ state: State }> = (props) => {
	const { state } = props;
	const { t } = useTranslation();
	const [dataDir, setDataDir] = useState("");

	useMount(async () => {
		setDataDir(await tauriDataDir());
	});

	const handleChange = async (isDefault = false) => {
		try {
			const nextDir = isDefault ? dataDir : await open({ directory: true });

			if (!isString(nextDir) || isEqualPath(nextDir)) return;

			state.spinning = true;

			const dirName = await moveData(getSaveDataDir(), nextDir);

			if (!dirName) return;

			globalStore.env.saveDataDir = joinPath(nextDir, dirName);

			state.spinning = false;

			emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);

			message.success(
				t("preference.data_backup.storage_path.hints.save_success"),
			);
		} catch (error: any) {
			state.spinning = false;

			message.error(error);
		}
	};

	const isEqualPath = (nextDir = dataDir) => {
		return isEqual(joinPath(nextDir, getSaveDataDirName()), getSaveDataDir());
	};

	return (
		<ProList header={t("preference.data_backup.storage_path.title")}>
			<ProListItem
				title={t("preference.data_backup.storage_path.label.storage_path")}
				description={
					<span
						className="hover:color-primary cursor-pointer break-all transition"
						onMouseDown={() => previewPath(getSaveDataDir())}
					>
						{getSaveDataDir()}
					</span>
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
							disabled={isEqualPath()}
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
