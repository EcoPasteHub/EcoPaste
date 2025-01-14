export const useOSKeyPress: typeof useKeyPress = (...args) => {
	const [keyFilter, handler, option] = args;

	useKeyPress(
		keyFilter,
		(event, key) => {
			const { metaKey, ctrlKey } = event;

			if ((metaKey && !isMac()) || (ctrlKey && isMac())) return;

			event.preventDefault();

			handler(event, key);
		},
		{ exactMatch: true, ...option },
	);
};
