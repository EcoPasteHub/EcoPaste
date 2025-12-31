import { fetch } from "@tauri-apps/plugin-http";

export interface TranslateConfig {
  apiBase: string;
  apiKey: string;
  model: string;
  targetLanguage: string;
  systemPrompt: string;
}

export interface TranslateResult {
  success: boolean;
  text: string;
  error?: string;
}

/**
 * 使用 OpenAI 兼容 API 进行翻译
 */
export async function translate(
  text: string,
  config: TranslateConfig,
): Promise<TranslateResult> {
  const { apiBase, apiKey, model, targetLanguage, systemPrompt } = config;

  if (!apiKey) {
    return {
      error: "API Key is not configured",
      success: false,
      text: "",
    };
  }

  if (!text.trim()) {
    return {
      error: "No text to translate",
      success: false,
      text: "",
    };
  }

  // 构建 API URL
  let url = apiBase.replace(/\/$/, "");
  if (!url.endsWith("/chat/completions")) {
    url = `${url}/v1/chat/completions`;
  }

  // 替换系统提示词中的变量
  const processedPrompt = systemPrompt.replace(
    /\$targetLanguage/g,
    targetLanguage,
  );

  try {
    const response = await fetch(url, {
      body: JSON.stringify({
        messages: [
          {
            content: processedPrompt,
            role: "system",
          },
          {
            content: text,
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

    const translatedText = data.choices[0]?.message?.content?.trim() || "";

    return {
      success: true,
      text: translatedText,
    };
  } catch (error) {
    return {
      error: `Request failed: ${error}`,
      success: false,
      text: "",
    };
  }
}
