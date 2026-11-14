import type { SelectProps } from "antd";
import { Select } from "antd";
import type { TFunction } from "i18next";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { LANGUAGE_OPTIONS } from "@/constants/languages";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Language } from "@/types/settings";
import { log } from "@/utils/log";

interface LanguageSwitcherProps {
  className?: string;
  disabled?: boolean;
  size?: SelectProps["size"];
  value?: Language;
  onChange?: (language: Language) => Promise<void> | void;
}

/**
 * 语言切换控件，复用偏好设置的语言文案与 antd Select 形态。
 */
const LanguageSwitcher: FC<LanguageSwitcherProps> = (props) => {
  const { value } = props;

  if (value !== void 0) {
    return <ControlledLanguageSwitcher {...props} />;
  }

  return <UncontrolledLanguageSwitcher {...props} />;
};

export default LanguageSwitcher;

const ControlledLanguageSwitcher: FC<LanguageSwitcherProps> = (props) => {
  const { value, onChange, ...rest } = props;
  const { t } = useTranslation("preferences");
  const currentLanguage = value as Language;

  const commitLanguageChange = async (nextLanguage: Language) => {
    if (nextLanguage === currentLanguage) return;

    await onChange?.(nextLanguage);
  };

  const handleSelectChange = async (nextLanguage: Language) => {
    try {
      await commitLanguageChange(nextLanguage);
    } catch (error) {
      log.warn("change language failed", error);
    }
  };

  return (
    <LanguageSelect
      {...rest}
      onChange={handleSelectChange}
      t={t}
      value={currentLanguage}
    />
  );
};

const UncontrolledLanguageSwitcher: FC<LanguageSwitcherProps> = (props) => {
  const { value: _value, onChange, ...rest } = props;
  const { t } = useTranslation("preferences");
  const settings = useSnapshot(settingsState);
  const currentLanguage = settings.appearance.language;

  const commitLanguageChange = async (nextLanguage: Language) => {
    if (nextLanguage === currentLanguage) return;

    if (onChange) {
      await onChange(nextLanguage);
      return;
    }

    await updateSettings({
      appearance: {
        language: nextLanguage,
      },
    });
  };

  const handleSelectChange = async (nextLanguage: Language) => {
    try {
      await commitLanguageChange(nextLanguage);
    } catch (error) {
      log.warn("change language failed", error);
    }
  };

  return (
    <LanguageSelect
      {...rest}
      onChange={handleSelectChange}
      t={t}
      value={currentLanguage}
    />
  );
};

interface LanguageSelectProps
  extends Omit<LanguageSwitcherProps, "onChange" | "value"> {
  t: TFunction<"preferences">;
  value: Language;
  onChange: (language: Language) => void;
}

const LanguageSelect: FC<LanguageSelectProps> = (props) => {
  const { className, disabled, size, t, value, onChange } = props;
  const options = LANGUAGE_OPTIONS.map((option) => {
    return {
      label: t(`schema.settings.appearance.language.options.${option.value}`),
      value: option.value,
    };
  });

  return (
    <Select<Language>
      className={className}
      disabled={disabled}
      onChange={onChange}
      options={options}
      size={size}
      value={value}
    />
  );
};
