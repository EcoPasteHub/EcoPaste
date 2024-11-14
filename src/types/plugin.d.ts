export type WindowLabel = (typeof WINDOW_LABEL)[keyof typeof WINDOW_LABEL];

export interface Metadata {
	size: number;
	isDir: boolean;
	isFile: boolean;
	isExist: boolean;
	name: string;
	extname: string;
}

export interface ReadImage {
	width: number;
	height: number;
	image: string;
}

export interface ClipboardPayload {
	type?: "text" | "rtf" | "html" | "image" | "files";
	group: "text" | "image" | "files";
	subtype?: "url" | "email" | "color" | "path";
	count: number;
	value: string;
	search: string;
	width?: number;
	height?: number;
}

export interface WindowsOCR {
	content: string;
	qr: Array<{
		bounds: Array<{ x: number; y: number }>;
		content: string;
	}>;
}
