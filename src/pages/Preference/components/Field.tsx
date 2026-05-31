import { ListBox, Select, Switch } from "@heroui/react";
import type { Key } from "react";

export const Toggle = ({
  isSelected,
  onChange,
}: {
  isSelected: boolean;
  onChange: (v: boolean) => void;
}) => (
  <Switch isSelected={isSelected} onChange={onChange}>
    <Switch.Control>
      <Switch.Thumb />
    </Switch.Control>
  </Switch>
);

interface SelectControlProps<T extends string> {
  value: T;
  onChange: (v: T) => void;
  options: { key: T; label: string }[];
  className?: string;
}

/**
 * 单选枚举的薄封装；空选区（onSelectionChange 给 null）保持原值不变。
 */
export const SelectControl = <T extends string>({
  value,
  onChange,
  options,
  className = "w-44",
}: SelectControlProps<T>) => (
  <Select
    className={className}
    onSelectionChange={(k: Key | null) => {
      if (k != null) onChange(k as T);
    }}
    selectedKey={value}
  >
    <Select.Trigger>
      <Select.Value />
      <Select.Indicator />
    </Select.Trigger>
    <Select.Popover>
      <ListBox>
        {options.map((o) => (
          <ListBox.Item id={o.key} key={o.key}>
            {o.label}
          </ListBox.Item>
        ))}
      </ListBox>
    </Select.Popover>
  </Select>
);
