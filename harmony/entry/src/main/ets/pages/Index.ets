import promptAction from '@ohos.promptAction';
import pasteboard from '@ohos.pasteboard';
import { SyncManager, SyncStatus } from '../services/SyncManager';
import { PreferenceManager, WebDavConfig } from '../services/PreferenceManager';
import { cleanToPlainText } from '../utils/TextFormatUtils';

@Entry
@Component
struct Index {
  @State syncStatus: SyncStatus = {
    online: false,
    syncing: false,
    currentText: '',
    sourceDevice: '',
    lastSyncTime: 0,
    message: '正在检查连接...'
  };

  @State config: WebDavConfig = {
    endpoint: '',
    username: '',
    password: '',
    remoteDir: '/EcoPaste/',
    autoSync: true,
    backgroundSync: true
  };

  @State isConfigOpen: boolean = false;
  @State isTesting: boolean = false;

  private syncManager: SyncManager = SyncManager.getInstance();
  private prefManager: PreferenceManager = PreferenceManager.getInstance();

  private statusCallback = (status: SyncStatus) => {
    this.syncStatus = status;
  };

  aboutToAppear(): void {
    this.syncManager.addListener(this.statusCallback);
    this.loadSavedConfig();
  }

  aboutToDisappear(): void {
    this.syncManager.removeListener(this.statusCallback);
  }

  async loadSavedConfig(): Promise<void> {
    const saved = await this.prefManager.loadConfig();
    this.config = saved;
    // 如果尚未配置服务器地址，默认展开配置卡片
    if (!saved.endpoint) {
      this.isConfigOpen = true;
    }
  }

  async saveConfig(): Promise<void> {
    await this.prefManager.saveConfig(this.config);
    this.syncManager.updateConfig(this.config);
    try {
      promptAction.showToast({
        message: '配置已保存并生效',
        duration: 2000
      });
    } catch (e) {
      // 忽略 toast 异常
    }
  }

  async testConnection(): Promise<void> {
    this.isTesting = true;
    await this.prefManager.saveConfig(this.config);
    this.syncManager.updateConfig(this.config);
    await this.syncManager.checkHealth();
    this.isTesting = false;

    try {
      promptAction.showToast({
        message: this.syncStatus.message,
        duration: 2500
      });
    } catch (e) {
      // 忽略 toast 异常
    }
  }

  async copyCurrentText(): Promise<void> {
    if (!this.syncStatus.currentText) {
      try {
        promptAction.showToast({ message: '暂无可复制内容', duration: 1500 });
      } catch (e) {
        // 忽略异常
      }
      return;
    }
    const cleanText = cleanToPlainText(this.syncStatus.currentText);
    const board = pasteboard.getSystemPasteboard();
    const data = pasteboard.createData(pasteboard.MIMETYPE_TEXT_PLAIN, cleanText);
    await board.setData(data);
    try {
      promptAction.showToast({
        message: '已复制纯文本到手机剪贴板，长按即可粘贴',
        duration: 2000
      });
    } catch (e) {
      // 忽略异常
    }
  }

  formatTime(ts: number): string {
    if (!ts) return '尚未同步';
    const now = new Date().getTime();
    const diff = Math.floor((now - ts) / 1000);
    if (diff < 10) return '刚刚';
    if (diff < 60) return `${diff} 秒前`;
    if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`;
    return new Date(ts).toLocaleTimeString();
  }

  build() {
    Column() {
      // 顶部导航栏
      Row() {
        Row() {
          Image($r('app.media.icon'))
            .width(36)
            .height(36)
            .borderRadius(8)
            .margin({ right: 10 })

          Column() {
            Text('EcoPaste 随行同步')
              .fontSize(19)
              .fontWeight(FontWeight.Bold)
              .fontColor('#182431')
            Text('电脑 ⇄ 手机 剪贴板实时无感同步')
              .fontSize(12)
              .fontColor('#7E868E')
          }
          .alignItems(HorizontalAlign.Start)
        }

        // 状态胶囊指示器
        Row() {
          Row()
            .width(8)
            .height(8)
            .borderRadius(4)
            .backgroundColor(this.syncStatus.syncing ? '#E6A23C' : (this.syncStatus.online ? '#52C41A' : '#F5222D'))
            .margin({ right: 6 })

          Text(this.syncStatus.syncing ? '同步中' : (this.syncStatus.online ? '已连接' : '未连接'))
            .fontSize(12)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.syncStatus.syncing ? '#E6A23C' : (this.syncStatus.online ? '#52C41A' : '#F5222D'))
        }
        .padding({ left: 10, right: 10, top: 4, bottom: 4 })
        .borderRadius(14)
        .backgroundColor(this.syncStatus.online ? '#F6FFED' : '#FFF1F0')
      }
      .width('100%')
      .justifyContent(FlexAlign.SpaceBetween)
      .padding({ left: 16, right: 16, top: 14, bottom: 12 })

      Scroll() {
        Column({ space: 14 }) {
          // 当前同步核心卡片
          Column() {
            Row() {
              Row({ space: 8 }) {
                Text(this.syncStatus.sourceDevice ? `📡 ${this.syncStatus.sourceDevice}` : '📡 同步剪贴板')
                  .fontSize(14)
                  .fontWeight(FontWeight.Bold)
                  .fontColor('#1F2329')

                Text('已自动清洗为纯文本')
                  .fontSize(10)
                  .fontColor('#0A59F7')
                  .backgroundColor('#E8F3FF')
                  .padding({ left: 6, right: 6, top: 2, bottom: 2 })
                  .borderRadius(4)
              }

              Text(this.formatTime(this.syncStatus.lastSyncTime))
                .fontSize(12)
                .fontColor('#8F959E')
            }
            .width('100%')
            .justifyContent(FlexAlign.SpaceBetween)
            .margin({ bottom: 10 })

            // 文本内容展示区
            Column() {
              if (this.syncStatus.currentText) {
                Text(this.syncStatus.currentText)
                  .fontSize(15)
                  .fontColor('#1F2329')
                  .lineHeight(22)
                  .maxLines(6)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .width('100%')
              } else {
                Text('暂无同步文本。\n当在电脑复制文字时，手机将自动收到并可直接粘贴。')
                  .fontSize(14)
                  .fontColor('#8F959E')
                  .lineHeight(20)
                  .width('100%')
                  .textAlign(TextAlign.Center)
                  .padding({ top: 20, bottom: 20 })
              }
            }
            .width('100%')
            .padding(14)
            .borderRadius(12)
            .backgroundColor('#F7F8FA')
            .margin({ bottom: 12 })

            // 动作按钮栏
            Row({ space: 10 }) {
              Button('📋 复制内容')
                .layoutWeight(1)
                .height(38)
                .fontSize(13)
                .fontWeight(FontWeight.Medium)
                .backgroundColor('#0A59F7')
                .onClick(() => {
                  this.copyCurrentText();
                })

              Button('⬇️ 从电脑拉取')
                .layoutWeight(1)
                .height(38)
                .fontSize(13)
                .fontWeight(FontWeight.Medium)
                .backgroundColor('#E8F3FF')
                .fontColor('#0A59F7')
                .onClick(() => {
                  this.syncManager.pullLatestFromCloud(true);
                })

              Button('⬆️ 推送本机')
                .layoutWeight(1)
                .height(38)
                .fontSize(13)
                .fontWeight(FontWeight.Medium)
                .backgroundColor('#F0F2F5')
                .fontColor('#1F2329')
                .onClick(() => {
                  this.syncManager.pushCurrentClipboardToCloud();
                })
            }
            .width('100%')

            // 状态说明提示文案
            Text(this.syncStatus.message)
              .fontSize(12)
              .fontColor('#8F959E')
              .margin({ top: 10 })
          }
          .width('100%')
          .padding(16)
          .borderRadius(16)
          .backgroundColor('#FFFFFF')
          .shadow({ radius: 8, color: '#0A000000', offsetY: 2 })

          // WebDAV 配置卡片
          Column() {
            Row() {
              Text('⚙️ WebDAV 服务器配置')
                .fontSize(15)
                .fontWeight(FontWeight.Bold)
                .fontColor('#1F2329')

              Text(this.isConfigOpen ? '收起 ▲' : '展开 ▼')
                .fontSize(13)
                .fontColor('#0A59F7')
                .onClick(() => {
                  this.isConfigOpen = !this.isConfigOpen;
                })
            }
            .width('100%')
            .justifyContent(FlexAlign.SpaceBetween)

            if (this.isConfigOpen) {
              Column({ space: 12 }) {
                Column() {
                  Text('服务器地址 (与电脑端填写的保持一致):')
                    .fontSize(12)
                    .fontColor('#646A73')
                    .margin({ bottom: 4 })
                    .alignSelf(ItemAlign.Start)
                  TextInput({ text: this.config.endpoint, placeholder: '如 http://192.168.1.100:5005/webdav/' })
                    .fontSize(14)
                    .height(40)
                    .borderRadius(8)
                    .backgroundColor('#F5F6F7')
                    .onChange((val: string) => {
                      this.config.endpoint = val;
                    })
                }

                Row({ space: 10 }) {
                  Column() {
                    Text('账号:')
                      .fontSize(12)
                      .fontColor('#646A73')
                      .margin({ bottom: 4 })
                      .alignSelf(ItemAlign.Start)
                    TextInput({ text: this.config.username, placeholder: '用户名' })
                      .fontSize(14)
                      .height(40)
                      .borderRadius(8)
                      .backgroundColor('#F5F6F7')
                      .onChange((val: string) => {
                        this.config.username = val;
                      })
                  }
                  .layoutWeight(1)

                  Column() {
                    Text('密码:')
                      .fontSize(12)
                      .fontColor('#646A73')
                      .margin({ bottom: 4 })
                      .alignSelf(ItemAlign.Start)
                    TextInput({ text: this.config.password, placeholder: '密码 / Token' })
                      .type(InputType.Password)
                      .fontSize(14)
                      .height(40)
                      .borderRadius(8)
                      .backgroundColor('#F5F6F7')
                      .onChange((val: string) => {
                        this.config.password = val;
                      })
                  }
                  .layoutWeight(1)
                }
                .width('100%')

                Column() {
                  Text('远端目录:')
                    .fontSize(12)
                    .fontColor('#646A73')
                    .margin({ bottom: 4 })
                    .alignSelf(ItemAlign.Start)
                  TextInput({ text: this.config.remoteDir, placeholder: '/EcoPaste/' })
                    .fontSize(14)
                    .height(40)
                    .borderRadius(8)
                    .backgroundColor('#F5F6F7')
                    .onChange((val: string) => {
                      this.config.remoteDir = val;
                    })
                }

                Row() {
                  Text('自动实时同步 (每 2.5 秒探测一次)')
                    .fontSize(14)
                    .fontColor('#1F2329')
                  Toggle({ type: ToggleType.Switch, isOn: this.config.autoSync })
                    .selectedColor('#0A59F7')
                    .onChange((isOn: boolean) => {
                      this.config.autoSync = isOn;
                    })
                }
                .width('100%')
                .justifyContent(FlexAlign.SpaceBetween)

                Row() {
                  Text('后台无感长时同步 (切出应用后继续)')
                    .fontSize(14)
                    .fontColor('#1F2329')
                  Toggle({ type: ToggleType.Switch, isOn: this.config.backgroundSync })
                    .selectedColor('#0A59F7')
                    .onChange((isOn: boolean) => {
                      this.config.backgroundSync = isOn;
                    })
                }
                .width('100%')
                .justifyContent(FlexAlign.SpaceBetween)

                Row({ space: 12 }) {
                  Button(this.isTesting ? '正在测试...' : '🔍 测试连接')
                    .layoutWeight(1)
                    .height(38)
                    .fontSize(13)
                    .backgroundColor('#F0F2F5')
                    .fontColor('#1F2329')
                    .onClick(() => {
                      this.testConnection();
                    })

                  Button('💾 保存并生效')
                    .layoutWeight(1)
                    .height(38)
                    .fontSize(13)
                    .backgroundColor('#0A59F7')
                    .onClick(() => {
                      this.saveConfig();
                    })
                }
                .width('100%')
                .margin({ top: 6 })
              }
              .margin({ top: 12 })
            }
          }
          .width('100%')
          .padding(16)
          .borderRadius(16)
          .backgroundColor('#FFFFFF')
          .shadow({ radius: 8, color: '#0A000000', offsetY: 2 })

          // 贴心说明卡片
          Column({ space: 8 }) {
            Text('💡 无感互通使用技巧')
              .fontSize(14)
              .fontWeight(FontWeight.Bold)
              .fontColor('#1F2329')
              .alignSelf(ItemAlign.Start)

            Text('1. 电脑复制 -> 手机粘贴：电脑端任意复制文本，手机端打开任意应用直接长按粘贴即可。')
              .fontSize(12)
              .fontColor('#646A73')
              .lineHeight(18)
              .alignSelf(ItemAlign.Start)

            Text('2. 手机复制 -> 电脑粘贴：手机端复制任何文本，电脑端无需任何多余操作，直接 Ctrl+V 即可粘贴。')
              .fontSize(12)
              .fontColor('#646A73')
              .lineHeight(18)
              .alignSelf(ItemAlign.Start)

            Text('3. 权限保证：已内置请求剪贴板读写与长时后台传输权限，确保无感顺畅体验。')
              .fontSize(12)
              .fontColor('#646A73')
              .lineHeight(18)
              .alignSelf(ItemAlign.Start)
          }
          .width('100%')
          .padding(14)
          .borderRadius(14)
          .backgroundColor('#F8F9FB')
        }
        .padding({ left: 16, right: 16, bottom: 24 })
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor('#F1F3F5')
  }
}
