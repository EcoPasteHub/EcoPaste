import { Button, Empty, Input, Spin, Tag } from "antd";
import { useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import Scrollbar from "@/components/Scrollbar";
import { copyTextToClipboard, pasteTextToClipboard } from "@/plugins/clipboard";
import { showWindow } from "@/plugins/window";
import { globalStore } from "@/stores/global";
import { scrollElementToCenter } from "@/utils/dom";
import { syncStore } from "@/utils/store";
import {
  fetchTenorFeaturedGifs,
  searchTenorGifs,
  type TenorGifItem,
} from "@/utils/tenor";
import { MainContext } from "../../../..";

const GIF_CATEGORIES = [
  {
    id: "featured",
    name: "热门",
  },
  {
    id: "happy",
    name: "开心",
    query: "happy",
  },
  {
    id: "love",
    name: "喜欢",
    query: "love",
  },
  {
    id: "funny",
    name: "搞笑",
    query: "funny",
  },
  {
    id: "wow",
    name: "震惊",
    query: "wow",
  },
  {
    id: "sad",
    name: "难过",
    query: "sad",
  },
  {
    id: "angry",
    name: "生气",
    query: "angry",
  },
] as const;

const GifPicker = () => {
  const { rootState } = useContext(MainContext);
  const { integration } = useSnapshot(globalStore);
  const { t } = useTranslation();
  const requestIdRef = useRef(0);
  const [activeCategoryId, setActiveCategoryId] = useState<string>(
    GIF_CATEGORIES[0].id,
  );
  const [apiKeyInput, setApiKeyInput] = useState(integration.tenorApiKey ?? "");
  const [items, setItems] = useState<TenorGifItem[]>([]);
  const [nextPos, setNextPos] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const apiKey = integration.tenorApiKey?.trim();
  const keyword = rootState.search?.trim();

  const activeCategory = useMemo(() => {
    return (
      GIF_CATEGORIES.find((item) => item.id === activeCategoryId) ??
      GIF_CATEGORIES[0]
    );
  }, [activeCategoryId]);

  useEffect(() => {
    setApiKeyInput(integration.tenorApiKey ?? "");
  }, [integration.tenorApiKey]);

  useEffect(() => {
    scrollElementToCenter(`gif-tab-${activeCategoryId}`);
  }, [activeCategoryId]);

  const fetchGifs = async (append = false, pos?: string) => {
    if (!apiKey) return;

    const requestId = ++requestIdRef.current;

    setError(void 0);
    if (append) {
      setLoadingMore(true);
    } else {
      setLoading(true);
    }

    try {
      const response = keyword
        ? await searchTenorGifs({
            apiKey,
            keyword,
            limit: 24,
            pos,
          })
        : activeCategory.id === "featured"
          ? await fetchTenorFeaturedGifs({
              apiKey,
              limit: 24,
              pos,
            })
          : await searchTenorGifs({
              apiKey,
              categoryQuery: activeCategory.query,
              limit: 24,
              pos,
            });

      if (requestId !== requestIdRef.current) return;

      setItems((current) => {
        if (!append) return response.items;

        const merged = new Map(current.map((item) => [item.id, item]));

        response.items.forEach((item) => {
          merged.set(item.id, item);
        });

        return [...merged.values()];
      });
      setNextPos(response.next);
    } catch (e) {
      if (requestId !== requestIdRef.current) return;

      setItems((current) => (append ? current : []));
      setNextPos(void 0);
      setError(
        e instanceof Error
          ? e.message
          : t("clipboard.gif.hints.request_failed"),
      );
    } finally {
      if (requestId !== requestIdRef.current) return;

      setLoading(false);
      setLoadingMore(false);
    }
  };

  useEffect(() => {
    setItems([]);
    setNextPos(void 0);
    setError(void 0);
    setLoading(false);
    setLoadingMore(false);
    requestIdRef.current += 1;

    if (!apiKey) return;

    const timer = window.setTimeout(
      () => {
        void fetchGifs();
      },
      keyword ? 350 : 0,
    );

    return () => {
      window.clearTimeout(timer);
    };
  }, [activeCategory.id, apiKey, keyword]);

  const handleSaveApiKey = async () => {
    const value = apiKeyInput.trim();
    if (!value) return;

    setSaving(true);

    try {
      globalStore.integration.tenorApiKey = value;
      await syncStore();
    } finally {
      setSaving(false);
    }
  };

  const handleGifSelect = async (item: TenorGifItem) => {
    try {
      await pasteTextToClipboard(item.gifUrl);
    } catch {
      await copyTextToClipboard(item.gifUrl);
    }
  };

  if (!apiKey) {
    return (
      <div className="mx-3 flex-1 rounded-md bg-color-2 p-4">
        <div className="font-medium text-base">
          {t("clipboard.gif.label.api_title")}
        </div>
        <div className="mt-2 text-color-2 text-sm">
          {t("clipboard.gif.hints.api_description")}
        </div>
        <Input.Password
          className="mt-4"
          onChange={(event) => {
            setApiKeyInput(event.target.value);
          }}
          placeholder={t("clipboard.gif.hints.api_placeholder")}
          value={apiKeyInput}
        />
        <div className="mt-4 flex gap-2">
          <Button
            loading={saving}
            onClick={() => void handleSaveApiKey()}
            type="primary"
          >
            {t("clipboard.gif.button.save_api")}
          </Button>
          <Button
            onClick={() => {
              showWindow("preference");
            }}
          >
            {t("clipboard.gif.button.open_preferences")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-3 flex flex-1 flex-col overflow-hidden rounded-md">
      <div className="px-2 pb-2">
        <div className="flex overflow-x-auto whitespace-nowrap">
          {GIF_CATEGORIES.map((item) => {
            const checked = item.id === activeCategory.id;

            return (
              <div id={`gif-tab-${item.id}`} key={item.id}>
                <Tag.CheckableTag
                  checked={checked}
                  className={checked ? "bg-primary!" : void 0}
                  onChange={() => {
                    setActiveCategoryId(item.id);
                  }}
                >
                  {t(`clipboard.gif.categories.${item.id}`)}
                </Tag.CheckableTag>
              </div>
            );
          })}
        </div>
      </div>

      <Scrollbar className="flex-1" offsetX={3}>
        <div className="px-2 pb-3">
          <div className="px-1 pb-3 text-color-2 text-sm">
            {keyword
              ? t("clipboard.gif.label.search_results", { keyword })
              : t(`clipboard.gif.categories.${activeCategory.id}`)}
          </div>

          {loading ? (
            <div className="flex h-50 items-center justify-center">
              <Spin />
            </div>
          ) : error ? (
            <div className="rounded-md bg-color-2 p-4">
              <div className="text-red-5 text-sm">{error}</div>
              <Button
                className="mt-3"
                onClick={() => void fetchGifs()}
                type="primary"
              >
                {t("clipboard.gif.button.retry")}
              </Button>
            </div>
          ) : items.length === 0 ? (
            <div className="flex h-50 items-center justify-center">
              <Empty description={t("clipboard.gif.hints.empty")} />
            </div>
          ) : (
            <>
              <div className="grid grid-cols-2 gap-3 pb-3 lg:grid-cols-3">
                {items.map((item) => {
                  return (
                    <button
                      className="group relative aspect-video overflow-hidden rounded-lg bg-color-2 text-left"
                      key={item.id}
                      onClick={() => void handleGifSelect(item)}
                      title={t("clipboard.gif.label.copy_gif_link")}
                      type="button"
                    >
                      <img
                        alt={item.title || "GIF"}
                        className="h-full w-full object-cover transition duration-200 group-hover:scale-103"
                        loading="lazy"
                        src={item.previewUrl}
                      />
                      <div className="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent p-2 text-white text-xs">
                        <div className="line-clamp-1">
                          {item.title || t("clipboard.gif.label.copy_gif_link")}
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>

              {nextPos && (
                <Button
                  block
                  loading={loadingMore}
                  onClick={() => void fetchGifs(true, nextPos)}
                >
                  {t("clipboard.gif.button.load_more")}
                </Button>
              )}
            </>
          )}

          <div className="px-1 pt-3 text-center text-color-2 text-xs">
            {t("clipboard.gif.label.powered_by")}{" "}
            <a href="https://tenor.com">Tenor</a>
          </div>
        </div>
      </Scrollbar>
    </div>
  );
};

export default GifPicker;
