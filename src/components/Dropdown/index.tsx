import { Dropdown as AntdDropdown, type DropdownProps } from "antd";
import type { FC, ReactElement, ReactNode } from "react";
import { cloneElement, isValidElement } from "react";
import { cn } from "@/utils/cn";

const DEFAULT_MENU_ICON_CLASS = "text-sm! shrink-0";

type AntdMenu = NonNullable<DropdownProps["menu"]>;
type AntdMenuItems = NonNullable<AntdMenu["items"]>;
type AntdMenuItem = NonNullable<AntdMenuItems[number]>;

export type DropdownMenuItem = AntdMenuItem & {
  children?: DropdownMenuItem[];
  icon?: ReactNode;
};

export type DropdownMenuItems = Array<DropdownMenuItem | null>;

export interface AppDropdownProps extends Omit<DropdownProps, "menu"> {
  menu?: Omit<NonNullable<DropdownProps["menu"]>, "items"> & {
    items?: DropdownMenuItems;
  };
}

/**
 * 统一渲染菜单图标尺寸；字符串图标按 UnoCSS 图标类名处理。
 */
const renderMenuIcon = (icon: ReactNode): ReactNode => {
  if (!icon) return icon;

  if (typeof icon === "string") {
    return <i aria-hidden className={cn(icon, DEFAULT_MENU_ICON_CLASS)} />;
  }

  if (!isValidElement<{ className?: string }>(icon)) {
    return (
      <i aria-hidden className={DEFAULT_MENU_ICON_CLASS}>
        {icon}
      </i>
    );
  }

  const element = icon as ReactElement<{ className?: string }>;

  return cloneElement(element, {
    className: cn(DEFAULT_MENU_ICON_CLASS, element.props.className),
  });
};

/**
 * 递归规范化 antd Menu items，并补齐图标默认样式。
 */
const normalizeMenuItems = (
  items: DropdownMenuItems | undefined,
): AntdMenu["items"] => {
  if (!items) return items;

  return items.map((item) => {
    if (!item) return item;

    const { children, icon, ...rest } = item;

    return {
      ...rest,
      children: normalizeMenuItems(children),
      icon: renderMenuIcon(icon),
    };
  }) as AntdMenu["items"];
};

/**
 * antd Dropdown 的统一封装：保留原生能力，并收口菜单项图标默认尺寸。
 */
const Dropdown: FC<AppDropdownProps> = (props) => {
  const { menu, ...rest } = props;
  const normalizedMenu = menu
    ? { ...menu, items: normalizeMenuItems(menu.items) }
    : menu;

  return <AntdDropdown menu={normalizedMenu} {...rest} />;
};

export default Dropdown;
