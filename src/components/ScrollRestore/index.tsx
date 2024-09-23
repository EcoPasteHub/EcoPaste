import type { FC, HTMLAttributes } from "react";
import { useBlocker } from "react-router-dom";

interface Position {
	[key: string]: ScrollToOptions;
}

const ScrollRestore: FC<HTMLAttributes<HTMLDivElement>> = (props) => {
	const { children, ...rest } = props;
	const { pathname } = useLocation();
	const containerRef = useRef<HTMLDivElement>(null);
	const scroll = useScroll(containerRef);
	const [position, setPosition] = useSessionStorageState<Position>("position");

	useBlocker(({ currentLocation }) => {
		setPosition((position) => ({
			...position,
			[currentLocation.pathname]: scroll as ScrollToOptions,
		}));

		return false;
	});

	useUpdateEffect(() => {
		if (!position) return;

		const options = position[pathname];

		containerRef.current?.scrollTo(options);
	}, [pathname]);

	return (
		<div {...rest} ref={containerRef}>
			{children}
		</div>
	);
};

export default ScrollRestore;
