import { Flex } from "antd";
import clsx from "clsx";
import { find, intersectionWith, isEqual, remove, some } from "lodash-es";
import type { FC, KeyboardEvent } from "react";
import { type Key, keys, modifierKeys, normalKeys } from "./keys";

interface ShortcutKeyProps {
	defaultValue?: Key[];
	onChange?: (value: Key[], oldValue: Key[]) => void;
}

interface State {
	value: Key[];
}

const ShortcutKey: FC<ShortcutKeyProps> = (props) => {
	const { defaultValue = [], onChange } = props;

	const containerRef = useRef<HTMLElement>(null);

	const state = useReactive<State>({
		value: defaultValue,
	});

	const handleFocus = () => {
		state.value = [];
	};

	const handleBlur = () => {
		if (!registrable()) {
			state.value = defaultValue;
		}

		onChange?.(state.value, defaultValue);
	};

	const handleKeyDown = (event: KeyboardEvent) => {
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

	return (
		<Flex
			ref={containerRef}
			tabIndex={0}
			align="center"
			gap="small"
			className="b color-3 b-color-1 hover:b-primary-5 focus:b-primary h-32 w-fit rounded-6 px-10 font-bold text-16 transition focus:shadow-[0_0_0_2px_rgba(5,145,255,0.1)] focus:outline-none"
			onFocus={handleFocus}
			onBlur={handleBlur}
			onKeyDown={handleKeyDown}
			onKeyUp={handleKeyUp}
		>
			<Flex gap={4}>
				{modifierKeys.map((item) => {
					const { key, symbol } = item;

					return (
						<span
							key={key}
							className={clsx("transition", {
								"color-primary": some(state.value, { key }),
							})}
						>
							{symbol}
						</span>
					);
				})}
			</Flex>

			{getNormalKey() && (
				<span className="color-primary">{getNormalKey().symbol}</span>
			)}
		</Flex>
	);
};

export default ShortcutKey;
