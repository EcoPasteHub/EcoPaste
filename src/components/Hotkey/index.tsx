import { useAutoAnimate } from "@formkit/auto-animate/react";
import { Flex } from "antd";
import { isEmpty, isEqual } from "arcdash";
import clsx from "clsx";
import { find, intersectionWith, map, remove, some } from "lodash-es";
import type { FC, KeyboardEvent, MouseEvent } from "react";
import Icon from "../Icon";
import { type Key, keys, modifierKeys, normalKeys } from "./keys";

interface HotkeyProps {
	defaultValue?: string;
	onChange?: (value: string) => void;
}

interface State {
	value: Key[];
}

const Hotkey: FC<HotkeyProps> = (props) => {
	const { defaultValue = "", onChange } = props;

	const { t } = useTranslation();

	const handleDefaultValue = () => {
		if (!defaultValue) return [];

		return defaultValue.split("+").map((shortcut) => find(keys, { shortcut })!);
	};

	const containerRef = useRef<HTMLElement>(null);
	const [animationParent] = useAutoAnimate();

	const state = useReactive<State>({
		value: handleDefaultValue(),
	});

	useMount(() => {
		animationParent(containerRef.current);
	});

	const isHovering = useHover(containerRef);

	const isFocusing = useFocusWithin(containerRef, {
		onFocus: () => {
			state.value = [];
		},
		onBlur: () => {
			if (!registrable()) {
				state.value = handleDefaultValue();
			}

			const changeValue = map(state.value, "shortcut").join("+");

			onChange?.(changeValue);
		},
	});

	const handleKeyDown = (event: KeyboardEvent) => {
		event.stopPropagation();
		event.preventDefault();

		const key = getEventKey(event);

		// 忽略大写锁定键、重复按键
		if (key === "CapsLock" || some(state.value, { key })) return;
		// 已经有普通按键就忽略其它的
		if (some(normalKeys, { key }) && getNormalKey()) return;

		const item = find(keys, { key });

		if (!item) return;

		state.value.push(item);

		if (registrable()) {
			containerRef.current?.blur();
		}
	};

	const handleKeyUp = async (event: KeyboardEvent) => {
		const key = getEventKey(event);

		remove(state.value, { key });
	};

	const getEventKey = (event: KeyboardEvent) => {
		let { key, code } = event;

		key = key.replace("Meta", "Command");

		const isModifierKey = some(modifierKeys, { key });

		return isModifierKey ? key : code;
	};

	const hasModifierKey = () => {
		return intersectionWith(state.value, modifierKeys, isEqual).length > 0;
	};

	const getNormalKey = () => {
		return intersectionWith(state.value, normalKeys, isEqual)[0];
	};

	const registrable = () => hasModifierKey() && getNormalKey();

	const handleClear = (event: MouseEvent) => {
		event.preventDefault();

		state.value = [];

		onChange?.("");
	};

	const renderContent = () => {
		if (isMac()) {
			return (
				<Flex ref={animationParent} gap="small" className="font-bold text-16">
					<Flex gap={4}>
						{modifierKeys.map((item) => {
							const { key, macosSymbol } = item;

							return (
								<span
									key={key}
									className={clsx("transition", {
										"color-primary": some(state.value, { key }),
									})}
								>
									{macosSymbol}
								</span>
							);
						})}
					</Flex>

					{getNormalKey() && (
						<span className="color-primary">{getNormalKey().symbol}</span>
					)}
				</Flex>
			);
		}

		return (
			<div ref={animationParent} className="whitespace-nowrap font-500 text-14">
				{isFocusing && isEmpty(state.value) ? (
					<span className="font-normal text-primary">
						{t("component.shortcut_key.hints.set_shortcut_key")}
					</span>
				) : isEmpty(state.value) ? (
					t("component.shortcut_key.hints.shortcut_key_not_set")
				) : (
					map(state.value, "symbol").join(" + ")
				)}
			</div>
		);
	};

	return (
		<Flex
			ref={containerRef}
			tabIndex={0}
			align="center"
			gap="small"
			className="antd-input group b-color-1 color-3 h-32 rounded-6 px-10"
			onKeyDown={handleKeyDown}
			onKeyUp={handleKeyUp}
		>
			{renderContent()}

			{isHovering && !isFocusing && !isEmpty(state.value) && (
				<Icon
					hoverable
					size={16}
					name="i-iconamoon:close-circle-1"
					onMouseDown={handleClear}
				/>
			)}
		</Flex>
	);
};

export default Hotkey;
