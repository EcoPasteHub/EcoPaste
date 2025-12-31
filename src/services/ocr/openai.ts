import { fetch } from "@tauri-apps/plugin-http";

export interface OcrConfig {
  apiBase: string;
  apiKey: string;
  model: string;
}

export interface OcrResult {
  success: boolean;
  text: string;
  error?: string;
}

/**
 * 使用 OpenAI 兼容 API 进行 OCR 识别
 */
export async function performOcr(
  imageBase64: string,
  config: OcrConfig,
): Promise<OcrResult> {
  const { apiBase, apiKey, model } = config;

  if (!apiKey) {
    return {
      error: "API Key is not configured",
      success: false,
      text: "",
    };
  }

  // 构建 API URL
  let url = apiBase.replace(/\/$/, "");
  if (!url.endsWith("/chat/completions")) {
    url = `${url}/v1/chat/completions`;
  }

  try {
    const response = await fetch(url, {
      body: JSON.stringify({
        max_tokens: 4096,
        messages: [
          {
            content: [
              {
                text: "请识别图片中的所有文字，只输出识别到的文字内容，不要添加任何解释、格式或标点符号修改。如果图片中没有文字，请回复[无文字]。",
                type: "text",
              },
              {
                image_url: {
                  url: `data:image/png;base64,${imageBase64}`,
                },
                type: "image_url",
              },
            ],
            role: "user",
          },
        ],
        model,
      }),
      headers: {
        Authorization: `Bearer ${apiKey}`,
        "Content-Type": "application/json",
      },
      method: "POST",
    });

    if (!response.ok) {
      const errorText = await response.text();
      return {
        error: `API Error (${response.status}): ${errorText}`,
        success: false,
        text: "",
      };
    }

    const data = (await response.json()) as {
      choices: Array<{
        message: {
          content: string;
        };
      }>;
    };

    const text = data.choices[0]?.message?.content?.trim() || "";

    return {
      success: true,
      text,
    };
  } catch (error) {
    return {
      error: `Request failed: ${error}`,
      success: false,
      text: "",
    };
  }
}
