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

export interface UploadResult {
  success: boolean;
  message?: string;
}

export class WebDavService {
  private endpoint: string = '';
  private authHeader: string = '';
  private remoteDir: string = 'EcoPaste';
  private lastFetchCode: number = 0;

  constructor(endpoint: string, username: string, password: string, remoteDir: string) {
    this.updateConfig(endpoint, username, password, remoteDir);
  }

  public getLastFetchCode(): number {
    return this.lastFetchCode;
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
      const bytes = textEncoder.encodeInto(`${username.trim()}:${password}`);
      const base64Str = helper.encodeToStringSync(bytes);
      this.authHeader = `Basic ${base64Str}`;
    } else {
      this.authHeader = '';
    }
  }

  private getHeaders(contentType: string = ''): Record<string, string> {
    const headers: Record<string, string> = {};
    if (contentType) {
      headers['Content-Type'] = contentType;
    }
    if (this.authHeader) {
      headers['Authorization'] = this.authHeader;
    }
    return headers;
  }

  public getLiveTextUrl(): string {
    const ep = this.endpoint.endsWith('/') ? this.endpoint.slice(0, -1) : this.endpoint;
    if (this.remoteDir && ep.endsWith('/' + this.remoteDir)) {
      return `${ep}/live_text.json`;
    }
    return `${this.endpoint}${this.remoteDir}/live_text.json`;
  }

  /**
   * 测试 WebDAV 连通性与认证凭据。
   * 采用 WebDAV 标准 OPTIONS 方法及目标文件双重探测，避免群晖/Apache 目录禁止 GET 导致 403 误报。
   */
  public async testConnection(): Promise<TestResult> {
    if (!this.endpoint) {
      return { success: false, message: '请填写 WebDAV 服务器地址' };
    }

    const client = http.createHttp();
    try {
      // 1. 优先使用 WebDAV 标准 OPTIONS 方法探测端点与鉴权凭据
      const optRes = await client.request(this.endpoint, {
        method: http.RequestMethod.OPTIONS,
        header: this.getHeaders(''),
        connectTimeout: 8000,
        readTimeout: 8000
      });

      const optCode = optRes.responseCode;
      if (optCode === 401) {
        return { success: false, message: '认证失败 (HTTP 401)，请检查用户名或密码' };
      }
      if (optCode === 403) {
        return { success: false, message: '权限不足 (HTTP 403)，请检查该账号是否有 WebDAV 访问权限' };
      }

      // 2. 若 OPTIONS 返回 2xx 成功，进一步探测 live_text.json 同步文件
      const liveUrl = this.getLiveTextUrl();
      const fileRes = await client.request(liveUrl, {
        method: http.RequestMethod.GET,
        header: this.getHeaders(''),
        connectTimeout: 8000,
        readTimeout: 8000
      });

      const fileCode = fileRes.responseCode;
      if (fileCode === 401) {
        return { success: false, message: '认证失败 (HTTP 401)，请检查用户名或密码' };
      }
      if (fileCode === 403) {
        return { success: false, message: '目录权限不足 (HTTP 403)，请在群晖中检查该文件夹权限' };
      }
      if (fileCode >= 200 && fileCode < 300) {
        return { success: true, message: 'WebDAV 连接成功！已检测到同步文件' };
      }
      if (fileCode === 404) {
        return { success: true, message: 'WebDAV 连接成功！同步文件尚未创建（电脑复制一次即可）' };
      }

      // 若 OPTIONS 成功但 GET 返回其他状态，只要 OPTIONS 成功说明鉴权通过
      if (optCode >= 200 && optCode < 300) {
        return { success: true, message: 'WebDAV 服务器连接成功！' };
      }

      return { success: false, message: `服务器返回状态码: HTTP ${optCode}` };
    } catch (err) {
      return { success: false, message: `无法连接服务器: ${err?.message || JSON.stringify(err)}` };
    } finally {
      client.destroy();
    }
  }

  /**
   * 上传手机复制的纯文本到 WebDAV
   */
  public async uploadLiveText(text: string): Promise<UploadResult> {
    if (!this.endpoint || !text.trim()) {
      return { success: false, message: '未配置服务器或文本为空' };
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
      if (code >= 200 && code < 300) {
        return { success: true };
      }
      if (code === 401) {
        return { success: false, message: 'WebDAV 认证失败 (HTTP 401)' };
      }
      if (code === 403) {
        return { success: false, message: 'WebDAV 写入权限不足 (HTTP 403)' };
      }
      return { success: false, message: `上传失败: HTTP ${code}` };
    } catch (err) {
      console.error('uploadLiveText error:', JSON.stringify(err));
      return { success: false, message: `上传异常: ${err?.message || JSON.stringify(err)}` };
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
        header: this.getHeaders(''),
        connectTimeout: 8000,
        readTimeout: 8000
      });

      this.lastFetchCode = res.responseCode;
      if (res.responseCode === 200 && res.result) {
        let text = '';
        if (typeof res.result === 'string') {
          text = res.result;
        } else {
          text = JSON.stringify(res.result);
        }
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

