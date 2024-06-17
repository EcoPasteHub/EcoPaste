import type { ClipboardPayload } from "./plugin";

export type TableName = "history";

export type HistoryGroup = "text" | "image" | "files";

export interface HistoryItem extends Partial<ClipboardPayload> {
	id?: number;
	value?: string;
	group?: HistoryGroup;
	createTime?: string;
	isCollected?: boolean;
}

export type TablePayload = HistoryItem;
