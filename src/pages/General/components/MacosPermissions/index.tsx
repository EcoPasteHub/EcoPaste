import Icon from "@/components/Icon";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";

const MacosPermissions = () => {
	const state = useReactive({
		accessibilityPermissions: false,
		fullDiskAccessPermissions: false,
	});

	useMount(async () => {
		checkAccessibility();

		checkFullDiskAccess();
	});

	const checkAccessibility = async () => {
		state.accessibilityPermissions = await requestAccessibilityPermissions();

		const check = async () => {
			const opened = await checkAccessibilityPermissions();

			if (!opened) {
				setTimeout(check, 1000);
			} else {
				state.accessibilityPermissions = true;
			}
		};

		check();
	};

	const checkFullDiskAccess = async () => {
		const opened = await checkFullDiskAccessPermissions();

		if (!opened) {
			const yes = await ask("需要开启完全磁盘访问权限来实现文件预览", {
				title: "需要开启完全磁盘访问权限",
				okLabel: "去系统设置开启",
				cancelLabel: "稍后开启",
			});

			if (!yes) return;

			requestFullDiskAccessPermissions();
		} else {
			state.fullDiskAccessPermissions = true;
		}
	};

	const renderStatus = (
		key: keyof typeof state,
		checkEvent: () => Promise<void>,
	) => {
		return (
			<div className="children:(inline-flex items-center gap-4 font-bold)">
				{state[key] ? (
					<div className="text-primary">
						<Icon name="i-lucide:circle-check" />
						已授权
					</div>
				) : (
					<div className="cursor-pointer text-danger" onMouseDown={checkEvent}>
						<Icon name="i-lucide:circle-arrow-right" />
						去授权
					</div>
				)}
			</div>
		);
	};

	return (
		isMac() && (
			<ProList header="权限设置">
				<ProListItem
					title="辅助功能访问权限"
					description="需要无障碍访问权限来操作剪切板内容"
				>
					{renderStatus("accessibilityPermissions", checkAccessibility)}
				</ProListItem>

				<ProListItem
					title="完全磁盘访问权限"
					description="需要完全磁盘访问权限来实现文件预览"
				>
					{renderStatus("fullDiskAccessPermissions", checkFullDiskAccess)}
				</ProListItem>
			</ProList>
		)
	);
};

export default MacosPermissions;
