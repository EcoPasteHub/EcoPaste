import LocalImage from "@/components/LocalImage";
import type { DatabaseSchemaHistory } from "@/types/database";
import type { WindowsOCR } from "@/types/plugin";
import { isString } from "es-toolkit";
import { isEmpty } from "es-toolkit/compat";
import type { FC } from "react";

const Image: FC<DatabaseSchemaHistory<"image">> = (props) => {
	const { id, value, search } = props;

	useMount(async () => {
		if (isString(search) || !clipboardStore.content.ocr) return;

		let ocrResult = await systemOCR(value);

		if (isWin) {
			const { content, qr } = JSON.parse(search) as WindowsOCR;

			if (isEmpty(qr)) {
				ocrResult = content;
			} else {
				ocrResult = qr[0].content;
			}
		}

		ocrResult ??= "";

		const db = await getDatabase();

		await db
			.updateTable("history")
			.set("search", ocrResult)
			.where("id", "=", id)
			.execute();
	});

	return <LocalImage src={value} className="max-h-21.5" />;
};

export default Image;
