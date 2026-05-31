import { convertFileSrc, invoke } from "@tauri-apps/api/core";
import { useEffect, useState } from "react";
import { TAURI_COMMAND } from "@/constants/commands";
import type { ClipboardItem } from "@/types/clipboard";
import { log } from "@/utils/log";

const formatSize = (bytes?: number | null): string => {
  if (!bytes) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

const ImageCard = ({ item }: { item: ClipboardItem }) => {
  const [src, setSrc] = useState<string | null>(null);
  const dims = item.width && item.height ? `${item.width}×${item.height}` : "";
  const size = formatSize(item.size);

  useEffect(() => {
    let cancelled = false;
    invoke<string>(TAURI_COMMAND.GET_CLIPBOARD_IMAGE_PATH, {
      fileName: item.content,
      thumbnail: true,
    })
      .then((path) => {
        if (cancelled) return;
        setSrc(convertFileSrc(path));
      })
      .catch((err) => log.error("get_clipboard_image_path failed", err));
    return () => {
      cancelled = true;
    };
  }, [item.content]);

  return (
    <div className="flex min-w-0 items-start gap-2">
      {src ? (
        <img
          alt=""
          className="b-split size-12 shrink-0 rounded border object-cover"
          src={src}
        />
      ) : (
        <div className="b-split size-12 shrink-0 rounded border bg-bg-elevated" />
      )}
      <div className="min-w-0 flex-1">
        <div className="c-text-tertiary text-xs">Image</div>
        <div className="c-text text-sm">
          {[dims, size].filter(Boolean).join(" · ")}
        </div>
      </div>
    </div>
  );
};

export default ImageCard;
