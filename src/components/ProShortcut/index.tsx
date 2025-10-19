import { Flex } from "antd";
import type { ListItemMetaProps } from "antd/es/list";
import { find, isEmpty, map, remove, some, split } from "es-toolkit/compat";
import type { FC, KeyboardEvent, MouseEvent } from "react";
import ProListItem from "../ProListItem";
import UnoIcon from "../UnoIcon";
import { type Key, keys, modifierKeys, standardKeys } from "./keyboard";

interface ProShortcutProps extends ListItemMetaProps {
  value?: string;
  isSystem?: boolean;
  onChange?: (value: string) => void;
}

interface State {
  value: Key[];
}

const ProShortcut: FC<ProShortcutProps> = (props) => {
  const { value = "", isSystem = true, onChange, ...rest } = props;

  const { t } = useTranslation();

  const separator = isSystem ? "+" : ".";
  const keyFiled = isSystem ? "tauriKey" : "hookKey";

  const parseValue = () => {
    if (!value) return [];

    return split(value, separator).map((key) => {
      return find(keys, { [keyFiled]: key })!;
    });
  };

  const state = useReactive<State>({
    value: parseValue(),
  });

  const containerRef = useRef<HTMLElement>(null);

  const isHovering = useHover(containerRef);

  const isFocusing = useFocusWithin(containerRef, {
    onBlur: () => {
      if (!isValidShortcut()) {
        state.value = parseValue();
      }

      handleChange();
    },
    onFocus: () => {
      state.value = [];
    },
  });

  const isValidShortcut = () => {
    if (state.value?.[0]?.eventKey?.startsWith("F")) {
      return true;
    }

    const hasModifierKey = some(state.value, ({ eventKey }) => {
      return some(modifierKeys, { eventKey });
    });
    const hasStandardKey = some(state.value, ({ eventKey }) => {
      return some(standardKeys, { eventKey });
    });

    return hasModifierKey && hasStandardKey;
  };

  const getEventKey = (event: KeyboardEvent) => {
    let { key, code } = event;

    key = key.replace("Meta", "Command");

    const isModifierKey = some(modifierKeys, { eventKey: key });

    return isModifierKey ? key : code;
  };

  const handleChange = () => {
    const nextValue = map(state.value, keyFiled).join(separator);

    onChange?.(nextValue);
  };

  const handleKeyDown = (event: KeyboardEvent) => {
    const eventKey = getEventKey(event);

    const matched = find(keys, { eventKey });
    const isInvalid = !matched;
    const isDuplicate = some(state.value, { eventKey });

    if (isInvalid || isDuplicate) return;

    state.value.push(matched);

    if (isValidShortcut()) {
      containerRef.current?.blur();
    }
  };

  const handleKeyUp = (event: KeyboardEvent) => {
    remove(state.value, { eventKey: getEventKey(event) });
  };

  const handleClear = (event: MouseEvent) => {
    event.preventDefault();

    state.value = [];

    handleChange();
  };

  return (
    <ProListItem {...rest}>
      <Flex
        align="center"
        className="antd-input b-color-1 h-8 min-w-32 rounded-md px-2.5"
        gap="small"
        justify="center"
        onKeyDown={handleKeyDown}
        onKeyUp={handleKeyUp}
        ref={containerRef}
        tabIndex={0}
      >
        {isEmpty(state.value) ? (
          isFocusing ? (
            t("component.shortcut_key.hints.press")
          ) : (
            t("component.shortcut_key.hints.click")
          )
        ) : (
          <div className="font-bold text-primary">
            {map(state.value, "symbol").join(" ")}
          </div>
        )}

        <UnoIcon
          className="absolute right-2 text-color-3"
          hidden={isFocusing || !isHovering || isEmpty(state.value)}
          hoverable
          name="i-iconamoon:close-circle-1"
          onMouseDown={handleClear}
          size={16}
        />
      </Flex>
    </ProListItem>
  );
};

export default ProShortcut;
