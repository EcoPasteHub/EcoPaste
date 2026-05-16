const TENOR_API_BASE_URL = "https://tenor.googleapis.com/v2";
const TENOR_CLIENT_KEY = "ecopaste_desktop";

interface TenorMediaFormat {
  url?: string;
}

interface TenorResult {
  id: string;
  content_description?: string;
  itemurl?: string;
  media_formats?: {
    gif?: TenorMediaFormat;
    mediumgif?: TenorMediaFormat;
    tinygif?: TenorMediaFormat;
    nanogif?: TenorMediaFormat;
  };
}

interface TenorResponse {
  next?: string;
  results?: TenorResult[];
}

export interface TenorGifItem {
  id: string;
  title: string;
  previewUrl: string;
  gifUrl: string;
  shareUrl?: string;
}

interface FetchTenorGifsOptions {
  apiKey: string;
  categoryQuery?: string;
  keyword?: string;
  limit?: number;
  pos?: string;
}

const getMediaUrl = (
  formats: TenorResult["media_formats"],
  keys: Array<keyof NonNullable<TenorResult["media_formats"]>>,
) => {
  for (const key of keys) {
    const value = formats?.[key]?.url;

    if (value) return value;
  }
};

const mapTenorResult = (item: TenorResult): TenorGifItem | undefined => {
  const gifUrl = getMediaUrl(item.media_formats, [
    "gif",
    "mediumgif",
    "tinygif",
  ]);
  const previewUrl = getMediaUrl(item.media_formats, [
    "tinygif",
    "nanogif",
    "mediumgif",
    "gif",
  ]);

  if (!gifUrl || !previewUrl) return;

  return {
    gifUrl,
    id: item.id,
    previewUrl,
    shareUrl: item.itemurl,
    title: item.content_description ?? "",
  };
};

const requestTenor = async (
  endpoint: "featured" | "search",
  options: FetchTenorGifsOptions,
) => {
  const { apiKey, categoryQuery, keyword, limit = 24, pos } = options;
  const params = new URLSearchParams({
    client_key: TENOR_CLIENT_KEY,
    key: apiKey,
    limit: String(limit),
    media_filter: "tinygif,gif,mediumgif,nanogif",
  });

  if (pos) {
    params.set("pos", pos);
  }

  const query = keyword?.trim() || categoryQuery?.trim();
  if (endpoint === "search" && query) {
    params.set("q", query);
  }

  const response = await fetch(`${TENOR_API_BASE_URL}/${endpoint}?${params}`);
  const payload = (await response.json()) as TenorResponse & {
    error?: string;
    message?: string;
  };

  if (!response.ok) {
    throw new Error(
      payload.error || payload.message || "Failed to fetch Tenor GIFs",
    );
  }

  return {
    items: (payload.results ?? [])
      .map(mapTenorResult)
      .filter(Boolean) as TenorGifItem[],
    next: payload.next,
  };
};

export const fetchTenorFeaturedGifs = (options: FetchTenorGifsOptions) => {
  return requestTenor("featured", options);
};

export const searchTenorGifs = (options: FetchTenorGifsOptions) => {
  return requestTenor("search", options);
};
