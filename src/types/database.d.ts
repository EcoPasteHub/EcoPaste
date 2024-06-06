export type TableName = "history";

export type HistoryType = "files" | "image" | "html" | "rtf" | "text";

export interface HistoryPayload {
	id?: number;
	type?: HistoryType;
	content?: string;
	createTime?: string;
	isFavorite?: boolean;
}

export type TablePayload = HistoryPayload;
