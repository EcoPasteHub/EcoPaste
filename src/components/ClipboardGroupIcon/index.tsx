import DOMPurify from "dompurify";
import type { CSSProperties, FC } from "react";
import type { ClipboardGroupIcon as ClipboardGroupIconValue } from "@/types/clipboard";
import { cn } from "@/utils/cn";

const GROUP_ICON_SIZE_CLASS = {
  base: "size-3.75",
  lg: "size-5",
  md: "size-4",
  sm: "size-3.5",
} as const;

const GROUP_ICON_MASK_VARIABLE = "--clipboard-group-icon-mask";

type ClipboardGroupIconSize = keyof typeof GROUP_ICON_SIZE_CLASS;
type ClipboardGroupIconMaskStyle = CSSProperties & {
  [GROUP_ICON_MASK_VARIABLE]: string;
};

interface ClipboardGroupIconProps {
  className?: string;
  disabled?: boolean;
  icon: ClipboardGroupIconValue;
  inheritColor?: boolean;
  selected?: boolean;
  size?: ClipboardGroupIconSize;
}

/**
 * 判断图标值是否为自定义 SVG 字符串。
 */
const isCustomSvgIcon = (icon: ClipboardGroupIconValue) => {
  return icon.trimStart().startsWith("<svg");
};

/**
 * 清洗用户上传的 SVG，避免脚本、外部对象和行内样式进入 DOM。
 */
const sanitizeGroupSvg = (icon: ClipboardGroupIconValue) => {
  return DOMPurify.sanitize(icon, {
    FORBID_ATTR: ["style"],
    FORBID_TAGS: ["foreignObject", "script"],
    USE_PROFILES: { svg: true },
  });
};

/**
 * 生成 SVG mask 的动态样式，内部颜色由 `bg-current` 接管。
 */
const buildGroupSvgMaskStyle = (
  icon: ClipboardGroupIconValue,
): ClipboardGroupIconMaskStyle => {
  const svg = sanitizeGroupSvg(icon);
  const mask = `url("data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}")`;

  return { [GROUP_ICON_MASK_VARIABLE]: mask };
};

/**
 * 统一渲染自定义分组图标，兼容 lets-icons 预设图标和自定义 SVG。
 */
const ClipboardGroupIcon: FC<ClipboardGroupIconProps> = (props) => {
  const {
    className,
    disabled = false,
    icon,
    inheritColor = false,
    selected = false,
    size = "sm",
  } = props;

  const stateClassName = cn({
    "text-ant-disabled": disabled && !inheritColor,
    "text-ant-light-solid": selected && !disabled && !inheritColor,
    "text-ant-secondary": !selected && !disabled && !inheritColor,
  });
  const sizeClassName = GROUP_ICON_SIZE_CLASS[size];

  if (isCustomSvgIcon(icon)) {
    return (
      <span
        aria-hidden
        className={cn(
          "inline-block shrink-0 bg-current [-webkit-mask-image:var(--clipboard-group-icon-mask)] [-webkit-mask-position:center] [-webkit-mask-repeat:no-repeat] [-webkit-mask-size:contain] [mask-image:var(--clipboard-group-icon-mask)] [mask-position:center] [mask-repeat:no-repeat] [mask-size:contain]",
          sizeClassName,
          stateClassName,
          className,
        )}
        style={buildGroupSvgMaskStyle(icon)}
      />
    );
  }

  return (
    <span
      aria-hidden
      className={cn(
        icon.trim(),
        "inline-block shrink-0",
        sizeClassName,
        stateClassName,
        className,
      )}
    />
  );
};

export default ClipboardGroupIcon;
