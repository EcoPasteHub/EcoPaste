import type { ClipboardPayload } from "./plugin";

export type TableName = "history";

export interface ClipboardItem extends ClipboardPayload {
	id: string;
	favorite: boolean;
	createTime: string;
	remark?: string;
}

export type TablePayload = Partial<ClipboardItem>;

export interface SelectPayload extends TablePayload {
	exact?: boolean;
}
