import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";
import { useSnapshot } from "valtio";

interface ScrollbarProps extends MacScrollbarProps {
	thumbSize?: number;
	offset?: number;
}

const Scrollbar = forwardRef<HTMLElement, ScrollbarProps>((props, ref) => {
	const { appearance } = useSnapshot(globalStore);

	const { thumbSize = 6, offset = 0, children, ...rest } = props;

	const containerRef = useRef<HTMLElement>(null);

	useImperativeHandle(ref, () => containerRef.current!);

	const getThumbStyle: MacScrollbarProps["thumbStyle"] = (horizontal) => {
		if (horizontal) {
			return {
				height: thumbSize,
				bottom: offset,
			};
		}

		return {
			width: thumbSize,
			right: offset,
		};
	};

	return (
		<MacScrollbar
			{...rest}
			ref={containerRef}
			skin={appearance.isDark ? "dark" : "light"}
			thumbStyle={getThumbStyle}
			trackStyle={() => ({ border: 0, "--ms-track-size": 0 })}
		>
			{children}
		</MacScrollbar>
	);
});

export default Scrollbar;
