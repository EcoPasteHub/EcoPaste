import { invoke } from "@tauri-apps/api";
import type { UpdateResult } from "@tauri-apps/api/updater";

/**
 * 检查 app 更新
 * @param joinBeta 是否参与测试
 */
export const checkUpdate = (joinBeta: boolean) => {
	return invoke<UpdateResult>(UPDATER_PLUGIN.CHECK_UPDATE, { joinBeta });
};
