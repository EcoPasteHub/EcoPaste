import { Select, Switch } from "antd";

/**
 * 受控开关：antd Switch 的 prop 命名是 checked / onChange。
 */
export const Toggle = ({
  isSelected,
  onChange,
}: {
  isSelected: boolean;
  onChange: (v: boolean) => void;
}) => <Switch checked={isSelected} onChange={onChange} />;

interface SelectControlProps<T extends string> {
  value: T;
  onChange: (v: T) => void;
  options: { key: T; label: string }[];
  className?: string;
}

/**
 * 单选枚举的薄封装；antd Select 受控走 value/onChange。
 */
export const SelectControl = <T extends string>({
  value,
  onChange,
  options,
  className = "w-44",
}: SelectControlProps<T>) => (
  <Select<T>
    className={className}
    onChange={(v) => onChange(v)}
    options={options.map((o) => ({ label: o.label, value: o.key }))}
    value={value}
  />
);
