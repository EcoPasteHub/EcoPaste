import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";

interface ScrollbarProps extends MacScrollbarProps {
	thumbSize?: number;
}

const Scrollbar = forwardRef<HTMLElement, ScrollbarProps>((props, ref) => {
	const { isDark } = useTheme();

	const { thumbSize = 6, children, ...rest } = props;

	const containerRef = useRef<HTMLElement>(null);

	useImperativeHandle(ref, () => containerRef.current!);

	const getThumbStyle: MacScrollbarProps["thumbStyle"] = (horizontal) => {
		if (horizontal) {
			return {
				height: thumbSize,
			};
		}

		return {
			width: thumbSize,
		};
	};

	return (
		<MacScrollbar
			{...rest}
			ref={containerRef}
			skin={isDark ? "dark" : "light"}
			thumbStyle={getThumbStyle}
			// @ts-ignore
			trackStyle={() => ({ "--ms-track-size": 0 })}
		>
			{children}
		</MacScrollbar>
	);
});

export default Scrollbar;
