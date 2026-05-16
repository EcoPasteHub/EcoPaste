import clsx from "clsx";
import type { FC, HTMLAttributes } from "react";

export interface UnoIconProps extends HTMLAttributes<HTMLElement> {
  name?: string;
  size?: number;
  color?: string;
  active?: boolean;
  hoverable?: boolean;
}

const UnoIcon: FC<UnoIconProps> = (props) => {
  const { name, className, size, color, active, hoverable, style, ...rest } =
    props;

  return (
    <i
      {...rest}
      className={clsx(name, className, "inline-flex", {
        "cursor-pointer transition hover:text-primary": hoverable,
        "text-primary": active,
      })}
      style={{
        color,
        height: size,
        width: size,
        ...style,
      }}
    />
  );
};

export default UnoIcon;
