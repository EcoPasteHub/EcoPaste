import { Flex } from "antd";
import clsx from "clsx";
import {
	find,
	intersectionWith,
	isEmpty,
	isEqual,
	map,
	remove,
	some,
} from "lodash-es";
import type { FC, KeyboardEvent, MouseEvent } from "react";
import Icon from "../Icon";
import ProListItem from "../ProListItem";
import { type Key, keys, modifierKeys, normalKeys } from "./keys";

interface ProShortcutProps {
	title: string;
	description?: string;
	value?: string;
	onChange?: (value: string) => void;
}

interface State {
	value: Key[];
}

const ProShortcut: FC<ProShortcutProps> = (props) => {
	const { title, description, value = "", onChange } = props;

	const { t } = useTranslation();

	const handleDefaultValue = () => {
		if (!value) return [];

		return value.split("+").map((shortcut) => find(keys, { shortcut })!);
	};

	const containerRef = useRef<HTMLElement>(null);

	const state = useReactive<State>({
		value: handleDefaultValue(),
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

	const registrable = () => {
		if (state.value.length === 1) {
			return /^F\d{1,2}$/.test(state.value[0].shortcut!);
		}

		return hasModifierKey() && getNormalKey();
	};

	const handleClear = (event: MouseEvent) => {
		event.preventDefault();

		state.value = [];

		onChange?.("");
	};

	const renderContent = () => {
		if (isMac()) {
			return (
				<Flex gap="small" className="font-bold text-16">
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
			<div className="font-500 text-14">
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
		<ProListItem title={title} description={description}>
			<Flex
				key={1}
				ref={containerRef}
				tabIndex={0}
				align="center"
				gap="small"
				className="antd-input group b-color-1 color-3 h-32 rounded-6 px-10"
				onKeyDown={handleKeyDown}
				onKeyUp={handleKeyUp}
			>
				{renderContent()}

				<Icon
					hoverable
					size={16}
					name="i-iconamoon:close-circle-1"
					hidden={isFocusing || !isHovering || isEmpty(state.value)}
					onMouseDown={handleClear}
				/>
			</Flex>
		</ProListItem>
	);
};

export default ProShortcut;
