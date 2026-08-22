import { useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";
import { TAURI_EVENT } from "@/constants/events";
import { WINDOW_LABEL } from "@/constants/windows";
import { useTauriListen } from "@/hooks/useTauriListen";

interface WindowVisibilityPayload {
  label: string;
  visible: boolean;
}

/**
 * Slide the bottom clipboard shelf up on show and down before Rust hides the window.
 */
export const useClipboardDockSlide = (enabled: boolean) => {
  const shouldReduceMotion = useReducedMotion();
  const [open, setOpen] = useState(enabled);

  useEffect(() => {
    setOpen(enabled);
  }, [enabled]);

  const handleVisibility = (event: { payload: WindowVisibilityPayload }) => {
    if (event.payload.label !== WINDOW_LABEL.CLIPBOARD) return;

    setOpen(enabled && event.payload.visible);
  };

  const handlePrepareHide = (event: { payload: WindowVisibilityPayload }) => {
    if (!enabled) return;
    if (event.payload.label !== WINDOW_LABEL.CLIPBOARD) return;

    setOpen(false);
  };

  useTauriListen<WindowVisibilityPayload>(
    TAURI_EVENT.WINDOW_VISIBILITY,
    handleVisibility,
  );
  useTauriListen<WindowVisibilityPayload>(
    TAURI_EVENT.WINDOW_PREPARE_HIDE,
    handlePrepareHide,
  );

  const transition = shouldReduceMotion
    ? { duration: 0 }
    : {
        damping: open ? 34 : 40,
        mass: 0.82,
        stiffness: open ? 480 : 560,
        type: "spring" as const,
      };

  return { open, transition };
};
