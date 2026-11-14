import { Empty } from "antd";
import type { TFunction } from "i18next";
import type { FC } from "react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Virtuoso } from "react-virtuoso";
import type {
  ClipboardPreviewFileEntry,
  ClipboardPreviewPayload,
} from "@/commands";
import AssetImage from "@/components/AssetImage";
import VirtuosoScroller, {
  type VirtuosoScrollerChildrenProps,
} from "@/components/VirtuosoScroller";
import { cn } from "@/utils/cn";
import { PREVIEW_TEXT_SOFT_WRAP_CHARS } from "../constants";

export interface PreviewContentProps {
  payload: ClipboardPreviewPayload | null;
}

export interface PreviewHeaderProps {
  payload: ClipboardPreviewPayload | null;
}

interface PayloadViewerProps {
  payload: ClipboardPreviewPayload;
}

interface FilePreviewRowProps {
  file: ClipboardPreviewFileEntry;
}

const TEXT_VIRTUOSO_COMPONENTS = {
  Footer: PreviewTextPadding,
  Header: PreviewTextPadding,
};

const FILES_VIRTUOSO_COMPONENTS = {
  Footer: PreviewFilesPadding,
  Header: PreviewFilesPadding,
};

/**
 * Content Viewer 顶部元信息区。
 */
export const PreviewHeader: FC<PreviewHeaderProps> = (props) => {
  const { payload } = props;
  const { t } = useTranslation(["preview", "clipboard"]);
  const title = payload ? previewTitle(t, payload) : t("title.loading");
  const meta = payload ? previewMeta(t, payload) : t("meta.contentViewer");
  const typeKey = payload ? (payload.subKind ?? payload.kind) : null;
  const typeLabel = typeKey ? t(`clipboard:types.${typeKey}`) : "";

  return (
    <div className="flex h-12 shrink-0 items-center justify-between gap-3 border-ant-border border-b px-4">
      <div className="min-w-0">
        <div className="truncate font-medium text-sm">{title}</div>
        <div className="truncate text-ant-secondary text-xs">{meta}</div>
      </div>

      {payload && (
        <span className="shrink-0 rounded-1 bg-ant-fill-secondary px-2 py-0.5 text-ant-secondary text-xs">
          {typeLabel}
        </span>
      )}
    </div>
  );
};

/**
 * 按 payload kind 分发到基础 viewer。
 */
export const PreviewContent: FC<PreviewContentProps> = (props) => {
  const { payload } = props;
  const { t } = useTranslation("preview");

  if (!payload) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty
          description={t("empty.content")}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

  if (payload.kind === "image") return <ImageViewer payload={payload} />;

  if (payload.kind === "files") return <FilesViewer payload={payload} />;

  return <TextViewer payload={payload} />;
};

/**
 * 文本预览：所有文本族内容都按纯文本虚拟行展示，避免长 HTML / RTF 构造大 DOM。
 */
const TextViewer: FC<PayloadViewerProps> = (props) => {
  const { payload } = props;
  const { t } = useTranslation("preview");
  const text = payload.text ?? "";
  const rows = useMemo(() => {
    return buildTextPreviewRows(text);
  }, [text]);

  if (text.length === 0) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty
          description={t("empty.text")}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

  return <VirtuosoScroller>{renderTextVirtuoso}</VirtuosoScroller>;

  function renderTextVirtuoso(props: VirtuosoScrollerChildrenProps) {
    const { scrollerRef } = props;

    return (
      <Virtuoso
        components={TEXT_VIRTUOSO_COMPONENTS}
        computeItemKey={computeTextRowKey}
        itemContent={renderTextRow}
        scrollerRef={scrollerRef}
        totalCount={rows.length}
      />
    );
  }

  function computeTextRowKey(index: number) {
    return index;
  }

  function renderTextRow(index: number) {
    const row = rows[index] ?? "";

    return (
      <div className="min-h-5.5 whitespace-pre px-4 font-mono text-xs leading-5.5">
        {row.length === 0 ? " " : row}
      </div>
    );
  }
};

/**
 * 图片预览：使用原图路径渲染，缺失时降级为空状态。
 */
const ImageViewer: FC<PayloadViewerProps> = (props) => {
  const { payload } = props;
  const { t } = useTranslation("preview");
  const imageWidth = payload.imageWidth ?? void 0;
  const imageHeight = payload.imageHeight ?? void 0;

  if (!payload.imagePath || !payload.imageExists) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty
          description={t("empty.imageMissing")}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 items-center justify-center p-4">
      <AssetImage
        alt={t("image.alt")}
        className="h-auto max-h-full max-w-full object-contain"
        draggable={false}
        height={imageHeight}
        src={payload.imagePath}
        width={imageWidth}
      />
    </div>
  );
};

/**
 * 文件预览：虚拟列表展示路径、文件名、存在状态与基础大小。
 */
const FilesViewer: FC<PayloadViewerProps> = (props) => {
  const { payload } = props;
  const { t } = useTranslation("preview");

  if (payload.files.length === 0) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty
          description={t("empty.files")}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

  return <VirtuosoScroller>{renderFilesVirtuoso}</VirtuosoScroller>;

  function renderFilesVirtuoso(props: VirtuosoScrollerChildrenProps) {
    const { scrollerRef } = props;
    const components =
      payload.totalFiles > payload.files.length
        ? {
            Footer: renderFilesFooter,
            Header: PreviewFilesPadding,
          }
        : FILES_VIRTUOSO_COMPONENTS;

    return (
      <Virtuoso
        components={components}
        computeItemKey={computeFileRowKey}
        itemContent={renderFileRow}
        scrollerRef={scrollerRef}
        totalCount={payload.files.length}
      />
    );
  }

  function computeFileRowKey(index: number) {
    return payload.files[index]?.path ?? index;
  }

  function renderFileRow(index: number) {
    const file = payload.files[index];
    if (!file) return <div className="h-10" />;

    return (
      <div className="px-2">
        <FilePreviewRow file={file} />
      </div>
    );
  }

  function renderFilesFooter() {
    return (
      <div className="px-4 py-2 text-ant-secondary text-xs">
        {t("file.shownCount", {
          shown: payload.files.length,
          total: payload.totalFiles,
        })}
      </div>
    );
  }
};

/**
 * 虚拟文本列表上下留白。
 */
function PreviewTextPadding() {
  return <div className="h-4" />;
}

/**
 * 虚拟文件列表顶部留白。
 */
function PreviewFilesPadding() {
  return <div className="h-2" />;
}

/**
 * 文件 viewer 的单行展示。
 */
const FilePreviewRow: FC<FilePreviewRowProps> = (props) => {
  const { file } = props;
  const { t } = useTranslation("preview");
  const kindLabel = file.isDir ? t("file.folder") : t("file.item");
  const sizeLabel = file.size === null ? kindLabel : formatBytes(file.size);

  return (
    <div
      className={cn(
        "flex min-h-10 items-center gap-2 rounded-1.5 px-2 py-1.5",
        { "opacity-50": !file.exists },
      )}
      title={file.path}
    >
      {file.iconPath ? (
        <AssetImage className="size-6 shrink-0" src={file.iconPath} />
      ) : (
        <i
          aria-hidden
          className="i-lucide:file size-5 shrink-0 text-ant-secondary"
        />
      )}

      <div className="min-w-0 flex-1">
        <div
          className={cn("truncate text-xs", {
            "line-through": !file.exists,
          })}
        >
          {file.name}
        </div>
        <div className="truncate text-ant-secondary text-xs">
          {file.exists ? file.path : t("file.missingPath")}
        </div>
      </div>

      <span className="shrink-0 text-ant-secondary text-xs">{sizeLabel}</span>
    </div>
  );
};

/**
 * 将长文本拆成虚拟行，超长单行按固定字符数软切块。
 */
function buildTextPreviewRows(text: string) {
  const rows: string[] = [];

  for (const line of text.split("\n")) {
    if (line.length === 0) {
      rows.push("");
      continue;
    }

    for (
      let start = 0;
      start < line.length;
      start += PREVIEW_TEXT_SOFT_WRAP_CHARS
    ) {
      rows.push(line.slice(start, start + PREVIEW_TEXT_SOFT_WRAP_CHARS));
    }
  }

  return rows;
}

/**
 * 生成 Content Viewer 标题。
 */
function previewTitle(
  t: TFunction<"preview">,
  payload: ClipboardPreviewPayload,
) {
  if (payload.kind === "files") {
    return t("title.files", { count: payload.totalFiles });
  }

  if (payload.kind === "image") {
    return t("title.image");
  }

  return t("title.text");
}

/**
 * 生成 Content Viewer 元信息。
 */
function previewMeta(
  t: TFunction<"preview">,
  payload: ClipboardPreviewPayload,
) {
  if (payload.kind === "files") {
    return t("meta.filesLoaded", { count: payload.files.length });
  }

  if (payload.kind === "image") {
    const dimensions =
      payload.imageWidth && payload.imageHeight
        ? `${payload.imageWidth} x ${payload.imageHeight}`
        : t("meta.unknownSize");
    const size = payload.size === null ? "" : ` · ${formatBytes(payload.size)}`;

    return `${dimensions}${size}`;
  }

  return t("meta.characters", {
    count: payload.size ?? payload.text?.length ?? 0,
  });
}

/**
 * 格式化字节大小为紧凑文本。
 */
function formatBytes(value: number) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }

  const fractionDigits = unitIndex === 0 || size >= 10 ? 0 : 1;

  return `${size.toFixed(fractionDigits)} ${units[unitIndex]}`;
}
