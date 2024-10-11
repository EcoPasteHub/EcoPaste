import { invoke } from "@tauri-apps/api/core";

/**
 * 获取鼠标坐标位置
 */
export const getMouseCoords = () => {
	return invoke<number[]>(MOUSE_PLUGIN.GET_MOUSE_COORDS);
};
