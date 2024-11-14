import type { ClipboardPayload } from "./plugin";

export type TableName = "history";

export interface HistoryTablePayload extends ClipboardPayload {
	id: string;
	favorite: boolean;
	createTime: string;
	note?: string;
}

export type TablePayload = Partial<HistoryTablePayload>;
