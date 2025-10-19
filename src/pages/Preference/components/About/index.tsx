import { AlipayOutlined, QqOutlined, WechatOutlined } from "@ant-design/icons";
import { getTauriVersion } from "@tauri-apps/api/app";
import { emit } from "@tauri-apps/api/event";
import { arch, version } from "@tauri-apps/plugin-os";
import { Avatar, Button, Image, message } from "antd";
import { writeText } from "tauri-plugin-clipboard-x-api";
import { useSnapshot } from "valtio";
import ProList from "@/components/ProList";
import ProListItem from "@/components/ProListItem";

const About = () => {
  const { appearance, env } = useSnapshot(globalStore);
  const { t } = useTranslation();
  const [visible, { toggle }] = useBoolean();
  const [imageSrc, setImageSrc] = useState("");

  const theme = useCreation(() => {
    return appearance.isDark ? "dark" : "light";
  }, [appearance.isDark]);

  const copyInfo = async () => {
    const { appName, appVersion, platform } = env;

    const info = {
      appName,
      appVersion,
      platform,
      platformArch: arch(),
      platformVersion: version(),
      tauriVersion: await getTauriVersion(),
    };

    await writeText(JSON.stringify(info, null, 2));

    message.success(t("preference.about.about_software.hints.copy_success"));
  };

  const previewImage = (src: string) => {
    setImageSrc(WEBSITE_LINK + src);
    toggle();
  };

  return (
    <ProList header={t("preference.about.about_software.title")}>
      <ProListItem
        avatar={<Avatar shape="square" size={44} src="/logo.png" />}
        description={`${t("preference.about.about_software.label.version")}v${env.appVersion}`}
        title={env.appName}
      >
        <Button
          onClick={() => {
            emit(LISTEN_KEY.UPDATE_APP, true);
          }}
          type="primary"
        >
          {t("preference.about.about_software.button.check_update")}
        </Button>
      </ProListItem>

      <ProListItem
        description={t("preference.about.about_software.hints.software_info")}
        title={t("preference.about.about_software.label.software_info")}
      >
        <Button onClick={copyInfo}>
          {t("preference.about.about_software.button.copy")}
        </Button>
      </ProListItem>

      <ProListItem
        description={<a href={GITHUB_LINK}>{GITHUB_LINK}</a>}
        title={t("preference.about.about_software.label.open_source_address")}
      >
        <Button danger href={GITHUB_ISSUES_LINK}>
          {t("preference.about.about_software.button.feedback_issue")}
        </Button>
      </ProListItem>

      <ProListItem title={t("preference.about.about_software.label.community")}>
        <Button
          className="hover:b-wechat!"
          icon={<WechatOutlined className="text-wechat" />}
          onClick={() => {
            previewImage(`/community/wechat-group-${theme}.png`);
          }}
        />
        <Button
          className="hover:b-qq!"
          icon={<QqOutlined className="text-qq" />}
          onClick={() => {
            previewImage(`/community/qq-group-${theme}.png`);
          }}
        />
      </ProListItem>

      <ProListItem title={t("preference.about.about_software.label.sponsor")}>
        <Button
          className="hover:b-wechat!"
          icon={<WechatOutlined className="text-wechat" />}
          onClick={() => {
            previewImage("/sponsor/wechat-pay.png");
          }}
        />
        <Button
          className="hover:b-alipay!"
          icon={<AlipayOutlined className="text-alipay" />}
          onClick={() => {
            previewImage("/sponsor/ali-pay.png");
          }}
        />
      </ProListItem>

      <Image
        hidden
        preview={{
          onVisibleChange: toggle,
          src: imageSrc,
          visible,
        }}
      />
    </ProList>
  );
};

export default About;
