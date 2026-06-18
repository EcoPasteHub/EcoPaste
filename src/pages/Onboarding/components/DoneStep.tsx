import { Result } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";

const DoneStep: FC = () => {
  const { t } = useTranslation("onboarding");

  return (
    <div className="flex min-h-0 flex-1 flex-col justify-center gap-10">
      <Result
        icon={
          <img
            alt={t("done.logoAlt")}
            className="mx-auto size-34 rounded-6"
            src="/logo.png"
          />
        }
        subTitle={t("done.description")}
        title={<span className="text-ant-text">{t("done.title")}</span>}
      />
    </div>
  );
};

export default DoneStep;
