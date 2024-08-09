export const useOSKeyPress: typeof useKeyPress = (
	keyFilter,
	handler,
	option,
) => {
	useKeyPress(
		keyFilter,
		(event, key) => {
			const { metaKey, ctrlKey } = event;

			if ((metaKey && !isMac()) || (ctrlKey && isMac())) return;

			event.preventDefault();

			handler(event, key);
		},
		option,
	);
};
