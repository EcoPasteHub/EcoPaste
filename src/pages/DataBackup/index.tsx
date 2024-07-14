import Icon from "@/components/Icon";
import { ask } from "@tauri-apps/api/dialog";
import { emit } from "@tauri-apps/api/event";
import { BaseDirectory, readTextFile } from "@tauri-apps/api/fs";
import { appDataDir } from "@tauri-apps/api/path";
import { Button, Card, Flex, Spin, message } from "antd";
import type { Store } from "antd/es/form/interface";

const DataBackup = () => {
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
			"导入的数据会完全覆盖所有的偏好设置以及所有的剪切板历史数据，你确定要导入吗？",
			{
				title: "导入须知",
				okLabel: "确定",
				cancelLabel: "取消",
				type: "error",
			},
		);

		if (!yes) return;

		await handleBackup("正在导入", importData, async (result) => {
			if (!result) return;

			const content = await readTextFile(STORE_FILE_NAME, {
				dir: BaseDirectory.AppData,
			});

			const parseContent = JSON.parse(content) as Store;

			Object.assign(globalStore, parseContent.globalStore);
			Object.assign(clipboardStore, parseContent.clipboardStore);

			emit(LISTEN_KEY.IMPORT_DATA);

			message.success("导入成功");

			initDatabase();
		});
	};

	const handleExport = async () => {
		const yes = await ask(
			"导出的数据包含所有的偏好设置以及所有的剪切板历史数据",
			{
				title: "导出须知",
				okLabel: "开始导出",
				cancelLabel: "取消",
			},
		);

		if (!yes) return;

		handleBackup("正在导出", exportData);
	};

	const renderList = [
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

	const openDir = async () => {
		previewPath(await appDataDir());
	};

	return (
		<Card
			title="导入和导出"
			extra={
				<Button ghost type="primary" onClick={openDir}>
					打开存储目录
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
							className="b b-dashed b-color-1 hover:b-primary h-102 w-102 cursor-pointer rounded-8 bg-3 transition hover:text-primary"
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
