import { Empty } from "antd";
import type { FC } from "react";
import type {
  ClipboardPreviewFileEntry,
  ClipboardPreviewPayload,
} from "@/commands";
import AssetImage from "@/components/AssetImage";
import SafeHtml from "@/components/SafeHtml";
import { cn } from "@/utils/cn";

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

/**
 * Content Viewer 顶部元信息区。
 */
export const PreviewHeader: FC<PreviewHeaderProps> = (props) => {
  const { payload } = props;
  const title = payload ? previewTitle(payload) : "正在加载";
  const meta = payload ? previewMeta(payload) : "Content Viewer";

  return (
    <div className="flex h-12 shrink-0 items-center justify-between gap-3 border-ant-border border-b px-4">
      <div className="min-w-0">
        <div className="truncate font-medium text-sm">{title}</div>
        <div className="truncate text-ant-secondary text-xs">{meta}</div>
      </div>

      {payload && (
        <span className="shrink-0 rounded-1 bg-ant-fill-secondary px-2 py-0.5 text-ant-secondary text-xs uppercase">
          {payload.subKind ?? payload.kind}
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

  if (!payload) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty
          description="暂无预览内容"
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
 * 文本预览：保留换行与空白，只提供只读滚动展示。
 */
const TextViewer: FC<PayloadViewerProps> = (props) => {
  const { payload } = props;
  const text = payload.text ?? "";

  if (text.length === 0) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty description="空文本" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </div>
    );
  }

  if (payload.subKind === "html") {
    return (
      <SafeHtml
        className="overflow-auto break-words p-4 text-xs leading-5.5"
        value={text}
      />
    );
  }

  return (
    <pre className="m-0 whitespace-pre-wrap break-words p-4 font-mono text-xs leading-5.5">
      {text}
    </pre>
  );
};

/**
 * 图片预览：使用原图路径渲染，缺失时降级为空状态。
 */
const ImageViewer: FC<PayloadViewerProps> = (props) => {
  const { payload } = props;

  if (!payload.imagePath || !payload.imageExists) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty
          description="图片文件不存在"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

  return (
    <div className="flex max-h-full items-center justify-center p-4">
      <AssetImage
        alt="clipboard image preview"
        className="max-h-full max-w-full object-contain"
        draggable={false}
        src={payload.imagePath}
      />
    </div>
  );
};

/**
 * 文件预览：紧凑列表展示路径、文件名、存在状态与基础大小。
 */
const FilesViewer: FC<PayloadViewerProps> = (props) => {
  const { payload } = props;

  if (payload.files.length === 0) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Empty description="无文件路径" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </div>
    );
  }

  return (
    <div className="overflow-auto p-2">
      <div className="flex flex-col gap-1">
        {payload.files.map((file) => {
          return <FilePreviewRow file={file} key={file.path} />;
        })}
      </div>

      {payload.totalFiles > payload.files.length && (
        <div className="px-2 py-2 text-ant-secondary text-xs">
          仅显示前 {payload.files.length} 项，共 {payload.totalFiles} 项
        </div>
      )}
    </div>
  );
};

/**
 * 文件 viewer 的单行展示。
 */
const FilePreviewRow: FC<FilePreviewRowProps> = (props) => {
  const { file } = props;
  const kindLabel = file.isDir ? "文件夹" : "文件";
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
        <span className="i-lucide:file size-5 shrink-0 text-ant-secondary" />
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
          {file.exists ? file.path : "路径已失效"}
        </div>
      </div>

      <span className="shrink-0 text-ant-secondary text-xs">{sizeLabel}</span>
    </div>
  );
};

/**
 * 生成 Content Viewer 标题。
 */
function previewTitle(payload: ClipboardPreviewPayload) {
  if (payload.kind === "files") {
    return `${payload.totalFiles} 个项目`;
  }

  if (payload.kind === "image") {
    return "图片预览";
  }

  return "文本预览";
}

/**
 * 生成 Content Viewer 元信息。
 */
function previewMeta(payload: ClipboardPreviewPayload) {
  if (payload.kind === "files") {
    return `${payload.files.length} 项已加载`;
  }

  if (payload.kind === "image") {
    const dimensions =
      payload.imageWidth && payload.imageHeight
        ? `${payload.imageWidth} x ${payload.imageHeight}`
        : "未知尺寸";
    const size = payload.size === null ? "" : ` · ${formatBytes(payload.size)}`;

    return `${dimensions}${size}`;
  }

  return `${payload.size ?? payload.text?.length ?? 0} 字符`;
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
