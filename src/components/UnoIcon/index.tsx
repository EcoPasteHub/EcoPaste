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
				"text-primary": active,
				"cursor-pointer transition hover:text-primary": hoverable,
			})}
			style={{
				color,
				width: size,
				height: size,
				...style,
			}}
		/>
	);
};

export default UnoIcon;
