/**
 * 字符串转 ArrayBuffer
 */
export const stringToArrayBuffer = (value: string) => {
	const buffer = new ArrayBuffer(value.length);

	const bufferView = new Uint8Array(buffer);

	for (let i = 0; i < value.length; i++) {
		bufferView[i] = value.charCodeAt(i);
	}

	return buffer;
};
