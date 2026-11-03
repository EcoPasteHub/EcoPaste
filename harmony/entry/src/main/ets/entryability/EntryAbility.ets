import UIAbility from '@ohos.app.ability.UIAbility';
import AbilityConstant from '@ohos.app.ability.AbilityConstant';
import Want from '@ohos.app.ability.Want';
import window from '@ohos.window';
import { SyncManager } from '../services/SyncManager';

export default class EntryAbility extends UIAbility {
  onCreate(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    console.info('EcoPaste EntryAbility onCreate');
  }

  onDestroy(): void {
    console.info('EcoPaste EntryAbility onDestroy');
  }

  onWindowStageCreate(windowStage: window.WindowStage): void {
    console.info('EcoPaste EntryAbility onWindowStageCreate');

    // 初始化剪贴板与 WebDAV 同步服务
    SyncManager.getInstance().init(this.context);

    windowStage.loadContent('pages/Index', (err, data) => {
      if (err.code) {
        console.error('Failed to load content. Cause:', JSON.stringify(err));
        return;
      }
      console.info('Succeeded in loading content.');
    });
  }

  onWindowStageDestroy(): void {
    console.info('EcoPaste EntryAbility onWindowStageDestroy');
  }

  onForeground(): void {
    console.info('EcoPaste EntryAbility onForeground');
    // 应用回到前台时，立即主动拉取一次最新云端剪贴板
    SyncManager.getInstance().pullLatestFromCloud(false);
  }

  onBackground(): void {
    console.info('EcoPaste EntryAbility onBackground');
  }
}
