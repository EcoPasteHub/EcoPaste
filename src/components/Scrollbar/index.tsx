import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";
import { useSnapshot } from "valtio";

const Scrollbar = forwardRef<HTMLElement, MacScrollbarProps>((props, ref) => {
	const { isDark } = useSnapshot(globalStore);

	const { children, ...rest } = props;

	const containerRef = useRef<HTMLElement>(null);

	useImperativeHandle(ref, () => containerRef.current!);

	return (
		<MacScrollbar ref={containerRef} skin={isDark ? "dark" : "light"} {...rest}>
			{children}
		</MacScrollbar>
	);
});

export default Scrollbar;
