import { listen } from "@tauri-apps/api/event";

export const useTauriListen = <T>(...args: Parameters<typeof listen<T>>) => {
  const unlistenRef = useRef(() => {});

  useMount(async () => {
    unlistenRef.current = await listen<T>(...args);
  });

  useUnmount(unlistenRef.current);
};
