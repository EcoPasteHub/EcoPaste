import { GithubFilled } from "@ant-design/icons";
import { openUrl } from "@tauri-apps/plugin-opener";
import { Button, Input, type InputRef } from "antd";
import { useRef } from "react";
import KeyHint from "@/components/KeyHint";
import { GITHUB_URL } from "@/constants/urls";

/**
 * 偏好设置顶部条：logo、搜索框（支持 ⌘F / Ctrl+F 聚焦）、主题/语言切换、GitHub 入口。
 */
const Header = () => {
  const searchRef = useRef<InputRef>(null);

  /**
   * 聚焦搜索框并选中已有内容，便于直接覆盖输入。
   */
  const focusSearch = () => {
    searchRef.current?.focus({ cursor: "all" });
  };

  /**
   * 通过 opener 插件打开 GitHub 主页（走系统默认浏览器，避免 webview 内跳转）。
   */
  const openGithub = () => {
    openUrl(GITHUB_URL);
  };

  return (
    <div
      className="flex items-center justify-between p-2"
      data-tauri-drag-region
    >
      <div className="flex items-center gap-2">
        <img alt="logo" className="size-6" src="/logo.png" />

        <span className="font-bold text-4.5">偏好设置</span>
      </div>

      <div className="flex gap-1">
        <Input
          className="w-40"
          placeholder="搜索设置项..."
          prefix={
            <KeyHint hintKey="F" onKeyPress={focusSearch}>
              <i className="i-lucide:search flex size-4" />
            </KeyHint>
          }
          ref={searchRef}
        />

        <Button>
          <i className="i-lucide:contrast" />
          <i className="i-lucide:moon" />
          <i className="i-lucide:sun" />
        </Button>

        <Button
          icon={<i className="i-lucide:languages text-4" />}
          type="text"
        />

        <Button
          className="text-4"
          icon={
            <KeyHint hintKey="G" onKeyPress={openGithub}>
              <GithubFilled />
            </KeyHint>
          }
          onClick={openGithub}
          type="text"
        />
      </div>
    </div>
  );
};

export default Header;
