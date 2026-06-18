import { Button } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { openExternalUrl } from "@/commands";
import { GITHUB_URL } from "@/constants/urls";
import { getMessageApi } from "@/utils/feedback";
import { APP_NAME_PLACEHOLDER } from "../constants";
import PreferenceCountTag from "./PreferenceCountTag";

interface PreferenceAboutPanelProps {
  appName: string;
  appVersion: string;
}

/**
 * 关于页：展示应用元信息、项目地址和赞赏二维码。
 */
const PreferenceAboutPanel: FC<PreferenceAboutPanelProps> = (props) => {
  const { appName, appVersion } = props;
  const { t } = useTranslation(["common", "preferences"]);
  const nameLabel = appName.length > 0 ? appName : APP_NAME_PLACEHOLDER;
  const versionLabel = appVersion.length > 0 ? `v${appVersion}` : "";

  const handleCheckUpdates = () => {
    getMessageApi().info(t("preferences:about.checkUpdatesUnavailable"));
  };

  const openGitHub = async () => {
    await openExternalUrl(GITHUB_URL);
  };

  return (
    <section className="w-full max-w-228 py-8">
      <div className="flex flex-col gap-8">
        <div className="flex min-w-0 flex-col gap-5 md:flex-row md:items-start md:justify-between">
          <div className="flex min-w-0 items-center gap-5">
            <img
              alt=""
              className="size-20 shrink-0 object-contain"
              draggable={false}
              src="/logo.png"
            />
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="m-0 font-semibold text-3xl text-ant-text leading-tight">
                  {nameLabel}
                </h2>
                {versionLabel.length > 0 ? (
                  <PreferenceCountTag className="text-ant-tertiary">
                    {versionLabel}
                  </PreferenceCountTag>
                ) : null}
              </div>
              <p className="m-0 mt-2 text-ant-secondary text-base leading-relaxed">
                {t("common:app.tagline")}
              </p>
            </div>
          </div>

          <div className="flex shrink-0 flex-wrap gap-2 md:pt-2">
            <Button
              icon={<i aria-hidden="true" className="i-lucide:refresh-cw" />}
              onClick={handleCheckUpdates}
            >
              {t("preferences:about.checkUpdates")}
            </Button>
            <Button
              icon={<i aria-hidden="true" className="i-lucide:github" />}
              onClick={openGitHub}
            >
              GitHub
            </Button>
          </div>
        </div>

        <div className="border-ant-border-secondary border-t pt-6">
          <div className="flex max-w-172 flex-col gap-5 rounded-2 border border-ant-border-secondary bg-ant-container px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex min-w-0 items-start gap-3">
              <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-ant-fill-tertiary text-ant-primary text-lg">
                <i aria-hidden="true" className="i-ph:hand-heart" />
              </span>
              <div className="min-w-0">
                <h3 className="m-0 font-semibold text-ant-text text-base leading-tight">
                  {t("preferences:about.sponsor")}
                </h3>
                <p className="m-0 mt-1 text-ant-secondary text-sm leading-relaxed">
                  {t("preferences:about.sponsorHint")}
                </p>
              </div>
            </div>
            <img
              alt={t("preferences:about.sponsorQrAlt")}
              className="size-32 self-center rounded-1 object-contain sm:self-auto"
              draggable={false}
              src="/sponsor-qr.png"
            />
          </div>
        </div>
      </div>
    </section>
  );
};

export default PreferenceAboutPanel;
