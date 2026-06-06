import { Button, message } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { openExternalUrl } from "@/commands";
import { GITHUB_URL } from "@/constants/urls";
import { APP_NAME_PLACEHOLDER } from "../constants";

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
    message.info(t("preferences:about.checkUpdatesUnavailable"));
  };

  const openGitHub = async () => {
    await openExternalUrl(GITHUB_URL);
  };

  return (
    <section className="flex min-h-full w-full items-center justify-center py-4">
      <div className="w-full max-w-212 overflow-hidden rounded-2 border border-ant-border-secondary bg-ant-container">
        <div className="grid gap-0 md:grid-cols-[minmax(0,1fr)_15rem]">
          <div className="flex min-h-58 flex-col justify-between px-6 py-6">
            <div className="flex min-w-0 items-start gap-4">
              <img
                alt=""
                className="size-16 shrink-0 object-contain"
                draggable={false}
                src="/logo.png"
              />
              <div className="min-w-0 flex-1 pt-1">
                <div className="flex flex-wrap items-center gap-3">
                  <h2 className="m-0 font-semibold text-2xl text-ant-text leading-tight">
                    {nameLabel}
                  </h2>
                  {versionLabel.length > 0 ? (
                    <span className="rounded-full border border-ant-border-secondary bg-ant-fill-quaternary px-3 py-1 text-ant-secondary text-xs leading-none">
                      {versionLabel}
                    </span>
                  ) : null}
                </div>
                <p className="m-0 mt-1 text-ant-secondary text-sm leading-relaxed">
                  {t("common:app.tagline")}
                </p>
              </div>
            </div>

            <div className="flex flex-wrap gap-2 border-ant-border-secondary border-t pt-4">
              <Button
                color="primary"
                icon={<i aria-hidden="true" className="i-lucide:refresh-cw" />}
                onClick={handleCheckUpdates}
                variant="outlined"
              >
                {t("preferences:about.checkUpdates")}
              </Button>
              <Button
                color="primary"
                icon={<i aria-hidden="true" className="i-lucide:github" />}
                onClick={openGitHub}
                variant="outlined"
              >
                GitHub
              </Button>
            </div>
          </div>

          <div className="flex min-h-58 flex-col items-center justify-center border-ant-border-secondary border-t bg-ant-fill-quaternary px-5 py-5 md:border-t-0 md:border-l">
            <div className="flex items-center gap-2">
              <i
                aria-hidden="true"
                className="i-ph:hand-heart text-ant-primary text-lg"
              />
              <h3 className="m-0 font-semibold text-ant-text text-sm leading-tight">
                {t("preferences:about.sponsor")}
              </h3>
            </div>
            <img
              alt={t("preferences:about.sponsorQrAlt")}
              className="mt-4 size-36 rounded-1 object-contain"
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
