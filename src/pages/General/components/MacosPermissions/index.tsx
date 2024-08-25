import Icon from "@/components/Icon";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";

const MacosPermissions = () => {
	const { t } = useTranslation();

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

			state.accessibilityPermissions = opened;

			if (opened) return;

			setTimeout(check, 1000);
		};

		check();
	};

	const checkFullDiskAccess = async () => {
		const opened = await checkFullDiskAccessPermissions();

		state.fullDiskAccessPermissions = opened;

		if (opened) return;

		const yes = await ask(
			t(
				"preference.settings.permission_settings.hints.confirm_full_disk_access",
			),
			{
				title: t(
					"preference.settings.permission_settings.label.confirm_full_disk_access",
				),
				okLabel: t(
					"preference.settings.permission_settings.button.confirm_full_disk_access",
				),
				cancelLabel: t(
					"preference.settings.permission_settings.button.cancel_full_disk_access",
				),
			},
		);

		if (!yes) return;

		requestFullDiskAccessPermissions();
	};

	const renderStatus = (
		key: keyof typeof state,
		mouseDownEvent: () => Promise<void>,
	) => {
		return (
			<div className="children:(inline-flex items-center gap-4 font-bold)">
				{state[key] ? (
					<div className="text-primary">
						<Icon name="i-lucide:circle-check" />
						{t("preference.settings.permission_settings.label.authorized")}
					</div>
				) : (
					<div
						className="cursor-pointer text-danger"
						onMouseDown={mouseDownEvent}
					>
						<Icon name="i-lucide:circle-arrow-right" />
						{t("preference.settings.permission_settings.button.authorize")}
					</div>
				)}
			</div>
		);
	};

	return (
		isMac() && (
			<ProList header={t("preference.settings.permission_settings.title")}>
				<ProListItem
					title={t(
						"preference.settings.permission_settings.label.accessibility_permissions",
					)}
					description={t(
						"preference.settings.permission_settings.hints.accessibility_permissions",
					)}
				>
					{renderStatus("accessibilityPermissions", checkAccessibility)}
				</ProListItem>

				<ProListItem
					title={t(
						"preference.settings.permission_settings.label.full_disk_access_permissions",
					)}
					description={t(
						"preference.settings.permission_settings.hints.full_disk_access_permissions",
					)}
				>
					{renderStatus("fullDiskAccessPermissions", checkFullDiskAccess)}
				</ProListItem>
			</ProList>
		)
	);
};

export default MacosPermissions;
