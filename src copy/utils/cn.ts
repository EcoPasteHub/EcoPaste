import clsx, { type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * 条件 className 工具：clsx 组合 + tailwind-merge 去重冲突类。
 * 后写的同族原子类（如 px-3 覆盖 px-2、bg-* 覆盖前一个 bg-*）会胜出，
 * 避免「父组件传的 className 被基础 className 反向覆盖」这种 footgun。
 * 推荐写法：cn("base", { "extra": cond }, propsClassName)
 */
export const cn = (...inputs: ClassValue[]): string => twMerge(clsx(inputs));
