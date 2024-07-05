import type { ClipboardPayload } from "./plugin";

export type TableName = "history";

export interface HistoryItem extends Partial<ClipboardPayload> {
	id?: number;
	group?: "text" | "image" | "files";
	createTime?: string;
	isCollected?: boolean;
}

export type TablePayload = HistoryItem;
