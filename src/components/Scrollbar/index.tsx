import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";
import { useSnapshot } from "valtio";

interface ScrollbarProps extends MacScrollbarProps {
	thumbSize?: number;
}

const Scrollbar = forwardRef<HTMLElement, ScrollbarProps>((props, ref) => {
	const { appearance } = useSnapshot(globalStore);

	const { thumbSize = 6, children, ...rest } = props;

	const containerRef = useRef<HTMLElement>(null);

	useImperativeHandle(ref, () => containerRef.current!);

	const getThumbStyle: MacScrollbarProps["thumbStyle"] = (horizontal) => {
		if (horizontal) {
			return {
				height: thumbSize,
				bottom: 0,
			};
		}

		return {
			width: thumbSize,
			right: 0,
		};
	};

	return (
		<MacScrollbar
			{...rest}
			ref={containerRef}
			skin={appearance.isDark ? "dark" : "light"}
			thumbStyle={getThumbStyle}
			// @ts-ignore
			trackStyle={() => ({ border: 0, "--ms-track-size": 0 })}
		>
			{children}
		</MacScrollbar>
	);
});

export default Scrollbar;
