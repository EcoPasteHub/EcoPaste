import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";

const Scrollbar = forwardRef<HTMLElement, MacScrollbarProps>((props, ref) => {
	const { isDark } = useTheme();

	const { children, ...rest } = props;

	const containerRef = useRef<HTMLElement>(null);

	useImperativeHandle(ref, () => containerRef.current!);

	const scrollBarWidth = "4px";

	return (
		<MacScrollbar
			ref={containerRef}
			skin={isDark ? "dark" : "light"}
			{...rest}
			thumbStyle={(horizontal) =>
				horizontal
					? {
							height: scrollBarWidth,
						}
					: {
							width: scrollBarWidth,
						}
			}
			trackStyle={(horizontal) =>
				horizontal
					? {
							height: scrollBarWidth,
						}
					: {
							width: scrollBarWidth,
						}
			}
		>
			{children}
		</MacScrollbar>
	);
});

export default Scrollbar;
