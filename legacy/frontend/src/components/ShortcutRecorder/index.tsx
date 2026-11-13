import { Input, type InputRef } from "antd";
import { AnimatePresence, motion } from "motion/react";
import {
  type FC,
  type KeyboardEvent,
  type MouseEvent,
  useEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { resumeGlobalShortcuts, suspendGlobalShortcuts } from "@/commands";
import { cn } from "@/utils/cn";
import { getMessageApi } from "@/utils/feedback";
import { log } from "@/utils/log";
import {
  buildShortcutFromEvent,
  buildShortcutPreviewFromEvent,
  formatRecordedShortcut,
  isRecordableShortcut,
  normalizeShortcutValue,
  resolveShortcutEventKey,
} from "@/utils/shortcut";

export interface ShortcutRecorderConflict {
  label: string;
  value: string;
}

interface ShortcutRecorderProps {
  className?: string;
  conflicts?: readonly ShortcutRecorderConflict[];
  disabled?: boolean;
  onChange?: (value: string) => void | Promise<void>;
  placeholder?: string;
  value?: string;
}

/**
 * 复用 Ant Design 只读 Input 视觉样式的通用快捷键录入器；
 * 外层容器捕获键盘事件，Input 只负责展示录入结果和聚焦态。
 */
const ShortcutRecorder: FC<ShortcutRecorderProps> = (props) => {
  const {
    className,
    conflicts = [],
    disabled = false,
    onChange,
    placeholder,
    value = "",
  } = props;
  const { t } = useTranslation("common");

  const inputRef = useRef<InputRef>(null);
  const committedValueRef = useRef(value);
  const draftValueRef = useRef(value);
  const recordingRef = useRef(false);
  const shortcutsSuspendedRef = useRef(false);
  const suspendPromiseRef = useRef<Promise<boolean> | null>(null);
  const [draft, setDraft] = useState(value);
  const [recording, setRecording] = useState(false);

  useEffect(() => {
    committedValueRef.current = value;

    if (recording) return;

    draftValueRef.current = value;
    setDraft(value);
  }, [recording, value]);

  useEffect(() => {
    return () => {
      if (!shortcutsSuspendedRef.current && !suspendPromiseRef.current) return;

      void resumeShortcutsAfterUnmount(
        suspendPromiseRef.current,
        shortcutsSuspendedRef.current,
      );
    };
  }, []);

  const displayValue = formatRecordedShortcut(draft);
  const showClearIcon = draft !== "" && !recording && !disabled;
  const inputPlaceholder = recording
    ? t("shortcutRecorder.press")
    : draft
      ? placeholder
      : t("shortcutRecorder.click");
  const fieldText = displayValue || inputPlaceholder;

  /**
   * 提交录入结果，并用 ref 去重避免按键完成和 blur 各保存一次。
   */
  const commit = async (nextValue: string) => {
    if (nextValue === committedValueRef.current) return;

    committedValueRef.current = nextValue;
    await onChange?.(nextValue);
  };

  /**
   * 同步更新草稿 ref 和 state，避免键盘事件与 React 渲染节奏错位。
   */
  const setDraftValue = (nextValue: string) => {
    draftValueRef.current = nextValue;
    setDraft(nextValue);
  };

  /**
   * 同步更新录制态 ref 和 React state，避免最后一次 keyup 清掉已提交结果。
   */
  const setRecordingState = (nextRecording: boolean) => {
    recordingRef.current = nextRecording;
    setRecording(nextRecording);
  };

  /**
   * 判断当前录入值是否和调用方传入的其它快捷键冲突。
   */
  const findShortcutConflict = (nextValue: string) => {
    const normalizedNextValue = normalizeShortcutValue(nextValue);
    if (!normalizedNextValue) return null;

    return (
      conflicts.find((conflict) => {
        return normalizeShortcutValue(conflict.value) === normalizedNextValue;
      }) ?? null
    );
  };

  /**
   * 冲突时提示占用来源并清空草稿，保持录入态等待用户重新按键。
   */
  const resetConflictedDraft = (
    conflict: ShortcutRecorderConflict,
    nextValue: string,
  ) => {
    getMessageApi().warning(
      t("shortcutRecorder.conflict", {
        label: conflict.label,
        shortcut: formatRecordedShortcut(nextValue),
      }),
    );
    setDraftValue("");
  };

  /**
   * 执行一次全局快捷键注销并记录结果；供 fire-and-forget 入口复用。
   */
  const runSuspendShortcuts = async () => {
    try {
      await suspendGlobalShortcuts();
      shortcutsSuspendedRef.current = true;
      suspendPromiseRef.current = null;

      return true;
    } catch (error) {
      log.error("suspend global shortcuts for recorder failed", error);
    }

    suspendPromiseRef.current = null;

    return false;
  };

  /**
   * 离开录入态后恢复全局快捷键，恢复时以 Rust 当前设置快照为准。
   */
  const resumeShortcuts = async () => {
    if (suspendPromiseRef.current) {
      await suspendPromiseRef.current;
    }
    if (!shortcutsSuspendedRef.current) return;

    shortcutsSuspendedRef.current = false;
    await resumeGlobalShortcuts();
  };

  /**
   * 进入录入态时立即触发注销；不阻塞 focus，但按键处理前会等待它完成。
   */
  const suspendShortcuts = () => {
    if (shortcutsSuspendedRef.current) return;
    if (suspendPromiseRef.current) return;

    suspendPromiseRef.current = runSuspendShortcuts();
  };

  /**
   * 清除当前快捷键并退出录入态，让 placeholder 回到「点击录制快捷键」。
   */
  const clearShortcut = async () => {
    setDraftValue("");
    setRecordingState(false);
    await commit("");
    await resumeShortcuts();
    inputRef.current?.blur();
  };

  /**
   * 聚焦后进入录制态，先清空展示等待用户按下新组合键。
   */
  const handleFocus = () => {
    if (disabled) return;
    if (recording) return;

    setDraftValue("");
    setRecordingState(true);
    suspendShortcuts();
  };

  /**
   * 失焦时校验草稿；无效录入回滚为外部当前值。
   */
  const handleBlur = async () => {
    if (!recordingRef.current) return;

    const draftValue = draftValueRef.current;
    const nextValue = isRecordableShortcut(draftValue) ? draftValue : value;
    const conflict = findShortcutConflict(nextValue);
    if (conflict) {
      resetConflictedDraft(conflict, nextValue);
      setRecordingState(false);
      setDraftValue(value);
      await resumeShortcuts();

      return;
    }

    setDraftValue(nextValue);
    setRecordingState(false);
    await commit(nextValue);
    await resumeShortcuts();
  };

  /**
   * 捕获冒泡上来的键盘事件，生成可交给 Tauri 注册的快捷键字面量。
   */
  const handleKeyDown = async (event: KeyboardEvent<HTMLFieldSetElement>) => {
    if (disabled) return;

    event.preventDefault();
    event.stopPropagation();

    if (suspendPromiseRef.current) {
      await suspendPromiseRef.current;
    }

    if (isClearKey(event)) {
      await clearShortcut();

      return;
    }

    if (event.key === "Escape") {
      await resumeShortcuts();
      inputRef.current?.blur();

      return;
    }

    const previewValue = buildShortcutPreviewFromEvent(event);
    if (!previewValue) return;

    setDraftValue(previewValue);

    const nextValue = buildShortcutFromEvent(event);
    if (!nextValue) return;
    if (!isRecordableShortcut(nextValue)) return;
    const conflict = findShortcutConflict(nextValue);
    if (conflict) {
      resetConflictedDraft(conflict, nextValue);

      return;
    }

    setRecordingState(false);
    await commit(nextValue);
    await resumeShortcuts();
    inputRef.current?.blur();
  };

  /**
   * 阻止录制中的按键继续冒泡到页面级快捷键处理器。
   */
  const handleKeyUp = (event: KeyboardEvent<HTMLFieldSetElement>) => {
    if (disabled) return;

    event.preventDefault();
    event.stopPropagation();

    if (!recordingRef.current) return;

    const releasedKey = resolveShortcutEventKey(event);
    if (!releasedKey) return;

    const nextValue = removeShortcutKey(
      draftValueRef.current,
      releasedKey.shortcutKey,
    );

    setDraftValue(nextValue);
  };

  /**
   * 鼠标点击清除图标时清空快捷键。
   */
  const handleClearClick = async () => {
    await clearShortcut();
  };

  return (
    <fieldset
      className={cn(
        "group/shortcut-recorder relative w-fit min-w-32 border-0 p-0",
        className,
      )}
      disabled={disabled}
      onBlur={handleBlur}
      onKeyDown={handleKeyDown}
      onKeyUp={handleKeyUp}
    >
      <span
        aria-hidden
        className="invisible block h-8 min-w-32 whitespace-pre px-3 text-center font-bold text-sm"
      >
        {displayValue}
      </span>
      <Input
        className="absolute inset-0 w-full min-w-0"
        classNames={{
          input: "text-transparent caret-transparent",
        }}
        disabled={disabled}
        onFocus={handleFocus}
        readOnly
        ref={inputRef}
      />
      <AnimatePresence initial={false} mode="wait">
        <motion.span
          animate={{ opacity: 1, scale: 1 }}
          className={cn(
            "pointer-events-none absolute inset-0 flex items-center justify-center whitespace-pre px-3 text-center text-sm",
            draft === "" ? "text-ant-tertiary" : "font-bold text-ant-primary",
          )}
          exit={{ opacity: 0, scale: 0.98 }}
          initial={{ opacity: 0, scale: 0.98 }}
          key={fieldText}
          transition={{ duration: 0.1, ease: "easeOut" }}
        >
          {fieldText}
        </motion.span>
      </AnimatePresence>
      {showClearIcon && (
        <button
          aria-label={t("shortcutRecorder.clear")}
          className="pointer-events-none absolute top-1/2 right-2 z-1 inline-flex size-4 -translate-y-1/2 cursor-pointer items-center justify-center border-0 bg-transparent p-0 text-ant-tertiary opacity-0 transition hover:text-ant-error group-hover/shortcut-recorder:pointer-events-auto group-hover/shortcut-recorder:opacity-100"
          data-shortcut-recorder-clear
          onClick={handleClearClick}
          onMouseDown={preventClearMouseDown}
          type="button"
        >
          <i className="i-lucide:circle-x size-3.5" />
        </button>
      )}
    </fieldset>
  );
};

export default ShortcutRecorder;

/**
 * 只有单独按删除键才视为清空；带修饰键时仍允许录入 Ctrl+Delete 等组合。
 */
function isClearKey(event: KeyboardEvent<HTMLFieldSetElement>) {
  if (event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) {
    return false;
  }

  return event.key === "Backspace" || event.key === "Delete";
}

/**
 * 清除按钮只处理清除，不让外层 mousedown 重新触发录入态。
 */
function preventClearMouseDown(event: MouseEvent<HTMLButtonElement>) {
  event.preventDefault();
  event.stopPropagation();
}

/**
 * 组件卸载时释放可能仍在路上的暂停请求，避免后端暂停计数泄漏。
 */
async function resumeShortcutsAfterUnmount(
  suspendPromise: Promise<boolean> | null,
  suspended: boolean,
) {
  let shouldResume = suspended;
  if (suspendPromise) {
    shouldResume = await suspendPromise;
  }
  if (!shouldResume) return;

  await resumeGlobalShortcuts();
}

/**
 * 从录入草稿里移除已经弹起的按键；录制态预览只展示仍按住的组合。
 */
function removeShortcutKey(value: string, shortcutKey: string) {
  return value
    .split("+")
    .filter((key) => {
      return key !== "" && key !== shortcutKey;
    })
    .join("+");
}
