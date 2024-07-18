import type { ClipboardPayload } from "./plugin";

export type TableName = "history";

export interface HistoryItem extends ClipboardPayload {
	id: number;
	group: "text" | "image" | "files";
	createTime: string;
	isCollected: boolean;
}

export type TablePayload = Partial<HistoryItem>;

export interface SelectPayload extends TablePayload {
	exact?: boolean;
}
