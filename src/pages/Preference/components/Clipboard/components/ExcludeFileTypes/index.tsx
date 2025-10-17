import ProSelect from "@/components/ProSelect";
import type { SelectProps } from "antd";
import { useSnapshot } from "valtio";

const ExcludeFileTypes = () => {
	const { excludeFiles } = useSnapshot(clipboardStore);
	const { t } = useTranslation();

	const options: SelectProps["options"] = [
		{
			value: ".txt",
			label: ".txt",
		},
		{
			value: ".pdf",
			label: ".pdf",
		},
		{
			value: ".mp3",
			label: ".mp3",
		},
		{
			value: ".mp4",
			label: ".mp4",
		},
		{
			value: ".avi",
			label: ".avi",
		},
		{
			value: ".png",
			label: ".png",
		},
		{
			value: ".jpg",
			label: ".jpg",
		},
		{
			value: ".gif",
			label: ".gif",
		},
		{
			value: ".doc",
			label: ".doc",
		},
		{
			value: ".docx",
			label: ".docx",
		},
		{
			value: ".xls",
			label: ".xls",
		},
		{
			value: ".xlsx",
			label: ".xlsx",
		},
		{
			value: ".ppt",
			label: ".ppt",
		},
		{
			value: ".pptx",
			label: ".pptx",
		},
		{
			value: ".zip",
			label: ".zip",
		},
		{
			value: ".rar",
			label: ".rar",
		},
	];

	return (
		<>
			<ProSelect
				title={t(
					"preference.clipboard.content_settings.label.excluded_file_types",
				)}
				description={t(
					"preference.clipboard.content_settings.hints.excluded_file_types",
				)}
				mode="tags"
				placeholder={t(
					"preference.clipboard.content_settings.hints.excluded_file_types_placeholder",
				)}
				expandWidth={180}
				options={options}
				defaultValue={excludeFiles}
				onChange={(value) => {
					clipboardStore.excludeFiles = value as string[];
				}}
			/>
		</>
	);
};

export default ExcludeFileTypes;
