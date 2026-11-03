import http from '@ohos.net.http';
import util from '@ohos.util';

export interface LivePayload {
  text: string;
  device: string;
  timestamp: number;
}

export interface TestResult {
  success: boolean;
  message: string;
}

export class WebDavService {
  private endpoint: string = '';
  private authHeader: string = '';
  private remoteDir: string = 'EcoPaste';

  constructor(endpoint: string, username: string, password: string, remoteDir: string) {
    this.updateConfig(endpoint, username, password, remoteDir);
  }

  public updateConfig(endpoint: string, username: string, password: string, remoteDir: string): void {
    let ep = endpoint.trim();
    if (ep && !ep.startsWith('http://') && !ep.startsWith('https://')) {
      ep = 'https://' + ep;
    }
    if (ep && !ep.endsWith('/')) {
      ep += '/';
    }
    this.endpoint = ep;

    let dir = remoteDir.trim();
    while (dir.startsWith('/')) {
      dir = dir.substring(1);
    }
    while (dir.endsWith('/')) {
      dir = dir.substring(0, dir.length - 1);
    }
    this.remoteDir = dir.length > 0 ? dir : 'EcoPaste';

    if (username || password) {
      const helper = new util.Base64Helper();
      const textEncoder = new util.TextEncoder();
      const bytes = textEncoder.encodeInto(`${username}:${password}`);
      const base64Str = helper.encodeToStringSync(bytes);
      this.authHeader = `Basic ${base64Str}`;
    } else {
      this.authHeader = '';
    }
  }

  private getHeaders(contentType: string = 'application/json'): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': contentType
    };
    if (this.authHeader) {
      headers['Authorization'] = this.authHeader;
    }
    return headers;
  }

  public getLiveTextUrl(): string {
    return `${this.endpoint}${this.remoteDir}/live_text.json`;
  }

  /**
   * 测试 WebDAV 连通性与认证凭据
   */
  public async testConnection(): Promise<TestResult> {
    if (!this.endpoint) {
      return { success: false, message: '请填写 WebDAV 服务器地址' };
    }

    const client = http.createHttp();
    try {
      const res = await client.request(this.endpoint, {
        method: http.RequestMethod.GET,
        header: this.getHeaders(),
        connectTimeout: 8000,
        readTimeout: 8000
      });

      const code = res.responseCode;
      if (code === 401 || code === 403) {
        return { success: false, message: `认证失败 (HTTP ${code})，请检查账号密码或群晖目录权限` };
      }
      if (code >= 200 && code < 400 || code === 404 || code === 405) {
        return { success: true, message: 'WebDAV 服务器连接成功！' };
      }
      return { success: false, message: `服务器返回状态码: HTTP ${code}` };
    } catch (err) {
      return { success: false, message: `无法连接服务器: ${err?.message || JSON.stringify(err)}` };
    } finally {
      client.destroy();
    }
  }

  /**
   * 上传手机复制的纯文本到 WebDAV
   */
  public async uploadLiveText(text: string): Promise<boolean> {
    if (!this.endpoint || !text.trim()) {
      return false;
    }

    const payload: LivePayload = {
      text,
      device: 'harmony',
      timestamp: new Date().getTime()
    };

    const client = http.createHttp();
    try {
      const res = await client.request(this.getLiveTextUrl(), {
        method: http.RequestMethod.PUT,
        header: this.getHeaders('application/json'),
        extraData: JSON.stringify(payload),
        connectTimeout: 10000,
        readTimeout: 10000
      });

      const code = res.responseCode;
      return code >= 200 && code < 300;
    } catch (err) {
      console.error('uploadLiveText error:', JSON.stringify(err));
      return false;
    } finally {
      client.destroy();
    }
  }

  /**
   * 从 WebDAV 获取远端剪贴板文本
   */
  public async fetchLiveText(): Promise<LivePayload | null> {
    if (!this.endpoint) {
      return null;
    }

    const client = http.createHttp();
    try {
      const res = await client.request(this.getLiveTextUrl(), {
        method: http.RequestMethod.GET,
        header: this.getHeaders(),
        connectTimeout: 8000,
        readTimeout: 8000
      });

      if (res.responseCode === 200 && res.result) {
        const text = typeof res.result === 'string' ? res.result : JSON.stringify(res.result);
        const data = JSON.parse(text) as LivePayload;
        return data;
      }
      return null;
    } catch (err) {
      return null;
    } finally {
      client.destroy();
    }
  }
}
