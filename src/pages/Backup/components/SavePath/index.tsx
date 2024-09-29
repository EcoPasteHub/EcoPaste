import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";
import { NodeIndexOutlined, ReloadOutlined } from "@ant-design/icons";
import { open } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { appLogDir, dataDir as tauriDataDir } from "@tauri-apps/api/path";
import { Button, Space, Tooltip, message } from "antd";
import { isEqual, isString } from "lodash-es";
import type { FC } from "react";
import type { State } from "../..";

const SavePath: FC<{ state: State }> = (props) => {
	const { state } = props;
	const { t } = useTranslation();
	const [dataDir, setDataDir] = useState("");
	const [logDir, setLogDir] = useState("");

	useMount(async () => {
		setDataDir(await tauriDataDir());
		setLogDir(await appLogDir());
	});

	const handleChange = async (isDefault = false) => {
		try {
			const dstDir = isDefault ? dataDir : await open({ directory: true });

			if (!isString(dstDir) || isEqualPath(dstDir)) return;

			const dstPath = joinPath(dstDir, getSaveDataDirName());

			state.spinning = true;

			await moveData(getSaveDataDir(), dstPath);

			globalStore.env.saveDataDir = dstPath;

			await wait();

			emit(LISTEN_KEY.REFRESH_CLIPBOARD_LIST);

			message.success(
				t("preference.data_backup.storage_settings.hints.change_success"),
			);

			state.spinning = false;
		} catch (error: any) {
			state.spinning = false;

			message.error(error);
		}
	};

	const isEqualPath = (dstDir = dataDir) => {
		const dstPath = joinPath(dstDir, getSaveDataDirName());

		return isEqual(dstPath, getSaveDataDir());
	};

	const description = (path = getSaveDataDir()) => {
		return (
			<span
				className="hover:color-primary cursor-pointer break-all transition"
				onMouseDown={() => previewPath(path, false)}
			>
				{joinPath(path)}
			</span>
		);
	};

	return (
		<ProList header={t("preference.data_backup.storage_settings.title")}>
			<ProListItem
				title={t(
					"preference.data_backup.storage_settings.label.data_storage_path",
				)}
				description={description()}
			>
				<Space.Compact>
					<Tooltip
						title={t(
							"preference.data_backup.storage_settings.hints.custom_path",
						)}
					>
						<Button
							icon={<NodeIndexOutlined />}
							onClick={() => handleChange()}
						/>
					</Tooltip>

					<Tooltip
						title={t(
							"preference.data_backup.storage_settings.hints.default_path",
						)}
					>
						<Button
							disabled={isEqualPath()}
							icon={<ReloadOutlined className="text-14!" />}
							onClick={() => handleChange(true)}
						/>
					</Tooltip>
				</Space.Compact>
			</ProListItem>

			<ProListItem
				title={t(
					"preference.data_backup.storage_settings.label.log_storage_path",
				)}
				description={description(logDir)}
			/>
		</ProList>
	);
};

export default SavePath;
