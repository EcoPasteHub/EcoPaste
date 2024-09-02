import type { ClipboardPayload } from "./plugin";

export type TableName = "history";

export interface ClipboardItem extends ClipboardPayload {
	id: number;
	group: "text" | "image" | "files";
	createTime: string;
	isCollected: boolean;
}

export type TablePayload = Partial<ClipboardItem>;

export interface SelectPayload extends TablePayload {
	exact?: boolean;
}
