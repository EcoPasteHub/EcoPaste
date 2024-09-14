import type { ClipboardPayload } from "./plugin";

export type TableName = "history";

export interface ClipboardItem extends ClipboardPayload {
	id: number;
	createTime: string;
	isCollected: boolean;
}

export type TablePayload = Partial<ClipboardItem>;

export interface SelectPayload extends TablePayload {
	exact?: boolean;
}
