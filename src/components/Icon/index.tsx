import clsx from "clsx";
import type { FC, HTMLAttributes } from "react";

export interface IconProps extends HTMLAttributes<HTMLElement> {
	name?: string;
	size?: number;
	hoverable?: boolean;
}

const Icon: FC<IconProps> = (props) => {
	const { name, className, size, hoverable, style, ...rest } = props;

	return (
		<i
			{...rest}
			className={clsx(name, className, "inline-flex", {
				"cursor-pointer transition hover:text-primary": hoverable,
			})}
			style={{
				...style,
				fontSize: `${size}px`,
			}}
		/>
	);
};

export default Icon;
