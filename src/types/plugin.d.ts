export interface Metadata {
	size: number;
	isDir: boolean;
	isFile: boolean;
	isExist: boolean;
	fileName: string;
}

export interface ReadImage {
	width: number;
	height: number;
	image: string;
}

export type ClipboardType = "text" | "rich-text" | "html" | "image" | "files";

export interface ClipboardPayload {
	type?: ClipboardType;
	size: number;
	value: string;
	search: string;
	width?: number;
	height?: number;
}

export interface WinOCR {
	content: string;
	qr: Array<{
		bounds: Array<{ x: number; y: number }>;
		content: string;
	}>;
}
