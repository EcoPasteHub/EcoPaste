import preferences from '@ohos.data.preferences';
import common from '@ohos.app.ability.common';

export interface WebDavConfig {
  endpoint: string;
  username: string;
  password: string;
  remoteDir: string;
  autoSync: boolean;
  backgroundSync: boolean;
}

const PREF_NAME = 'ecopaste_preferences';
const KEY_ENDPOINT = 'webdav_endpoint';
const KEY_USERNAME = 'webdav_username';
const KEY_PASSWORD = 'webdav_password';
const KEY_REMOTE_DIR = 'webdav_remote_dir';
const KEY_AUTO_SYNC = 'auto_sync';
const KEY_BG_SYNC = 'bg_sync';

export class PreferenceManager {
  private static instance: PreferenceManager | null = null;
  private pref: preferences.Preferences | null = null;

  public static getInstance(): PreferenceManager {
    if (!PreferenceManager.instance) {
      PreferenceManager.instance = new PreferenceManager();
    }
    return PreferenceManager.instance;
  }

  public async init(context: common.Context): Promise<void> {
    try {
      this.pref = await preferences.getPreferences(context, PREF_NAME);
    } catch (err) {
      console.error('PreferenceManager init failed:', JSON.stringify(err));
    }
  }

  public async loadConfig(): Promise<WebDavConfig> {
    if (!this.pref) {
      return {
        endpoint: '',
        username: '',
        password: '',
        remoteDir: '/EcoPaste/',
        autoSync: true,
        backgroundSync: true
      };
    }

    const endpoint = (await this.pref.get(KEY_ENDPOINT, '')) as string;
    const username = (await this.pref.get(KEY_USERNAME, '')) as string;
    const password = (await this.pref.get(KEY_PASSWORD, '')) as string;
    const remoteDir = (await this.pref.get(KEY_REMOTE_DIR, '/EcoPaste/')) as string;
    const autoSync = (await this.pref.get(KEY_AUTO_SYNC, true)) as boolean;
    const backgroundSync = (await this.pref.get(KEY_BG_SYNC, true)) as boolean;

    return {
      endpoint,
      username,
      password,
      remoteDir,
      autoSync,
      backgroundSync
    };
  }

  public async saveConfig(config: WebDavConfig): Promise<void> {
    if (!this.pref) {
      return;
    }

    await this.pref.put(KEY_ENDPOINT, config.endpoint);
    await this.pref.put(KEY_USERNAME, config.username);
    await this.pref.put(KEY_PASSWORD, config.password);
    await this.pref.put(KEY_REMOTE_DIR, config.remoteDir);
    await this.pref.put(KEY_AUTO_SYNC, config.autoSync);
    await this.pref.put(KEY_BG_SYNC, config.backgroundSync);
    await this.pref.flush();
  }
}

