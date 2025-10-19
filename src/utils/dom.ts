/**
 * 将指定 ID 的元素平滑滚动到视图中央
 */
export const scrollElementToCenter = (id?: string) => {
	if (!id) return;

	const activeElement = document.getElementById(id);

	activeElement?.scrollIntoView({
		behavior: "smooth",
		block: "center",
		inline: "center",
	});
};
