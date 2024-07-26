import Icon from "@/components/Icon";
import { emit } from "@tauri-apps/api/event";
import { BaseDirectory, readTextFile } from "@tauri-apps/api/fs";
import { appDataDir } from "@tauri-apps/api/path";
import { Button, Card, Flex, Spin, message } from "antd";
import type { Store } from "antd/es/form/interface";

const DataBackup = () => {
	const { t } = useTranslation();

	const state = useReactive({
		spinning: false,
		tip: "",
	});

	const toggleSpinning = () => {
		state.spinning = !state.spinning;
	};

	const handleBackup = async (
		tip: string,
		cb: () => Promise<unknown>,
		onSuccess?: (result: unknown) => void,
	) => {
		try {
			state.tip = tip;

			toggleSpinning();

			const result = await cb();

			toggleSpinning();

			onSuccess?.(result);
		} catch (error: any) {
			toggleSpinning();

			message.error(error);
		}
	};

	const handleImport = async () => {
		const yes = await ask(
			t("preference.data_backup.import_export.hints.import_confirm"),
			{
				title: t(
					"preference.data_backup.import_export.label.import_confirm_title",
				),
				okLabel: t("preference.data_backup.import_export.button.confirm"),
				cancelLabel: t("preference.data_backup.import_export.button.cancel"),
				type: "warning",
			},
		);

		if (!yes) return;

		await handleBackup(
			t("preference.data_backup.import_export.hints.importing"),
			importData,
			async (result) => {
				if (!result) return;

				const content = await readTextFile(STORE_FILE_NAME, {
					dir: BaseDirectory.AppData,
				});

				const parseContent = JSON.parse(content) as Store;

				Object.assign(globalStore, parseContent.globalStore);
				Object.assign(clipboardStore, parseContent.clipboardStore);

				emit(LISTEN_KEY.IMPORT_DATA);

				message.success(
					t("preference.data_backup.import_export.hints.import_success"),
				);

				initDatabase();
			},
		);
	};

	const handleExport = async () => {
		const yes = await ask(
			t("preference.data_backup.import_export.hints.export_confirm"),
			{
				title: t(
					"preference.data_backup.import_export.label.export_confirm_title",
				),
				okLabel: t(
					"preference.data_backup.import_export.button.confirm_export",
				),
				cancelLabel: t("preference.data_backup.import_export.button.cancel"),
			},
		);

		if (!yes) return;

		handleBackup(
			t("preference.data_backup.import_export.hints.exporting"),
			exportData,
		);
	};

	const renderList = [
		{
			label: t("preference.data_backup.import_export.button.import"),
			icon: "i-hugeicons:database-import",
			event: handleImport,
		},
		{
			label: t("preference.data_backup.import_export.button.export"),
			icon: "i-hugeicons:database-export",
			event: handleExport,
		},
	];

	const openDir = async () => {
		previewPath(await appDataDir());
	};

	return (
		<Card
			title={t("preference.data_backup.import_export.title")}
			extra={
				<Button ghost type="primary" onClick={openDir}>
					{t("preference.data_backup.import_export.button.open_dir")}
				</Button>
			}
		>
			<Flex
				gap="middle"
				onClick={(event) => {
					event.stopPropagation();
					event.preventDefault();
				}}
			>
				{renderList.map((item) => {
					const { label, icon, event } = item;

					return (
						<Flex
							key={label}
							vertical
							align="center"
							justify="center"
							gap="small"
							className="b b-dashed b-color-1 hover:b-primary h-102 w-102 cursor-pointer rounded-8 bg-3 px-8 text-center transition hover:text-primary"
							onClick={event}
						>
							<Icon name={icon} size={26} />
							{label}
						</Flex>
					);
				})}
			</Flex>

			<Spin spinning={state.spinning} tip={state.tip} fullscreen />
		</Card>
	);
};

export default DataBackup;
