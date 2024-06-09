export type TableName = "history";

export type HistoryType = "files" | "image" | "html" | "rtf" | "text";
export type HistoryGroup = "text" | "image" | "files";

export interface HistoryItem {
	id?: number;
	type?: HistoryType;
	group?: HistoryGroup;
	content?: string;
	createTime?: string;
	isCollected?: boolean;
}

export type TablePayload = HistoryItem;
