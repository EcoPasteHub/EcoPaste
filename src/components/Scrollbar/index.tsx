import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";

const Scrollbar = forwardRef<HTMLElement, MacScrollbarProps>((props, ref) => {
	const { isDark } = useTheme();

	const { children, ...rest } = props;

	const containerRef = useRef<HTMLElement>(null);

	useImperativeHandle(ref, () => containerRef.current!);

	const getThumbStyle: MacScrollbarProps["thumbStyle"] = (horizontal) => {
		const SCROLLBAR_THUMB_SIZE = 6;

		if (horizontal) {
			return {
				height: SCROLLBAR_THUMB_SIZE,
			};
		}

		return {
			width: SCROLLBAR_THUMB_SIZE,
		};
	};

	const getTrackStyle: MacScrollbarProps["trackStyle"] = () => {
		return {
			border: 0,
			background: "unset",
		};
	};

	return (
		<MacScrollbar
			{...rest}
			ref={containerRef}
			skin={isDark ? "dark" : "light"}
			thumbStyle={getThumbStyle}
			trackStyle={getTrackStyle}
		>
			{children}
		</MacScrollbar>
	);
});

export default Scrollbar;
