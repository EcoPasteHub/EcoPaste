import { emit } from "@tauri-apps/api/event";
import { Flex } from "antd";
import clsx from "clsx";
import { MacScrollbar } from "mac-scrollbar";
import { useSnapshot } from "valtio";
import UnoIcon from "@/components/UnoIcon";
import UpdateApp from "@/components/UpdateApp";
import About from "./components/About";
import Clipboard from "./components/Clipboard";
import General from "./components/General";
import History from "./components/History";
import Shortcut from "./components/Shortcut";

const Preference = () => {
  const { t } = useTranslation();
  const { app, shortcut, appearance } = useSnapshot(globalStore);
  const [activeKey, setActiveKey] = useState("clipboard");
  const contentRef = useRef<HTMLElement>(null);

  const { createTray } = useTray();

  useMount(async () => {
    createTray();

    const autostart = await isAutostart();

    if (!autostart && !app.silentStart) {
      showWindow();
    }
  });

  // 监听全局配置项变化
  useSubscribe(globalStore, () => handleStoreChanged());

  // 监听剪贴板配置项变化
  useSubscribe(clipboardStore, () => handleStoreChanged());

  // 监听快捷键切换窗口显隐
  useRegister(toggleWindowVisible, [shortcut.preference]);

  // 配置项变化通知其它窗口和本地存储
  const handleStoreChanged = () => {
    emit(LISTEN_KEY.STORE_CHANGED, { clipboardStore, globalStore });

    saveStore();
  };

  const menuItems = useCreation(() => {
    return [
      {
        content: <Clipboard />,
        icon: "i-lucide:clipboard-list",
        key: "clipboard",
        label: t("preference.menu.title.clipboard"),
      },
      {
        content: <History />,
        icon: "i-lucide:history",
        key: "history",
        label: t("preference.menu.title.history"),
      },
      {
        content: <General />,
        icon: "i-lucide:bolt",
        key: "general",
        label: t("preference.menu.title.general"),
      },
      {
        content: <Shortcut />,
        icon: "i-lucide:keyboard",
        key: "shortcut",
        label: t("preference.menu.title.shortcut"),
      },
      // {
      //   content: <Backup />,
      //   icon: "i-lucide:database-backup",
      //   key: "backup",
      //   label: t("preference.menu.title.backup"),
      // },
      {
        content: <About />,
        icon: "i-lucide:info",
        key: "about",
        label: t("preference.menu.title.about"),
      },
    ];
  }, [appearance.language]);

  const handleMenuClick = (key: string) => {
    setActiveKey(key);

    raf(() => {
      contentRef.current?.scrollTo({ behavior: "smooth", top: 0 });
    });
  };

  return (
    <Flex className="h-screen">
      <Flex
        className={clsx("h-full w-50 p-3", [isMac ? "pt-8" : "bg-color-1"])}
        data-tauri-drag-region
        gap="small"
        vertical
      >
        {menuItems.map((item) => {
          const { key, label, icon } = item;

          return (
            <Flex
              align="center"
              className={clsx(
                "cursor-pointer rounded-lg p-3 p-r-0 text-color-2 transition hover:bg-color-4",
                {
                  "bg-primary! text-white!": activeKey === key,
                },
              )}
              gap="small"
              key={key}
              onClick={() => handleMenuClick(key)}
            >
              <UnoIcon name={icon} size={20} />

              <span className="font-bold">{label}</span>
            </Flex>
          );
        })}
      </Flex>

      <MacScrollbar
        className="h-full flex-1 bg-color-2 p-4"
        data-tauri-drag-region
        ref={contentRef}
        skin={appearance.isDark ? "dark" : "light"}
      >
        {menuItems.map((item) => {
          const { key, content } = item;

          return (
            <div hidden={key !== activeKey} key={key}>
              {content}
            </div>
          );
        })}
      </MacScrollbar>

      <UpdateApp />
    </Flex>
  );
};

export default Preference;
