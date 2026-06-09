import { useMount } from "ahooks";
import { Spin } from "antd";
import { motion } from "motion/react";
import { type FC, useRef, useState } from "react";
import { useSnapshot } from "valtio";
import {
  type ClipboardPreviewState,
  getClipboardPreviewState,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import { settingsState } from "@/stores/settings";
import { cn } from "@/utils/cn";
import { log } from "@/utils/log";
import { cacheKey } from "./cache";
import { PreviewContent, PreviewHeader } from "./components/PreviewContent";
import PreviewContentTransition from "./components/PreviewContentTransition";
import {
  PREVIEW_CONNECTOR_VARIANTS,
  PREVIEW_PANEL_TRANSITION,
  PREVIEW_PANEL_VARIANTS,
} from "./constants";
import { resolveConnector } from "./geometry";
import { usePreviewPayload, usePreviewRenderState } from "./hooks";
import {
  rectStyle,
  resolveDynamicPanelRect,
  resolveEffectivePanelSize,
  resolveMeasurePanelStyle,
} from "./layout";
import { hasMeasuredPanelSize, useMeasuredPanelSize } from "./measurement";
import { usePreviewMotion } from "./motion";

const EMPTY_RECT = {
  height: 1,
  left: 0,
  top: 0,
  width: 1,
};
const EMPTY_POINT = { x: 0, y: 0 };
const EMPTY_CONNECTOR = {
  control1: EMPTY_POINT,
  control2: EMPTY_POINT,
  path: "M 0 0 C 0 0 0 0 0 0",
  source: EMPTY_POINT,
  sourceDot: EMPTY_POINT,
  sourceSide: "right",
  target: EMPTY_POINT,
  targetDot: EMPTY_POINT,
  targetSide: "left",
} as const;

/**
 * 系统级剪贴板预览窗口。
 * 预览窗口自身常驻透明 overlay，按 `itemId + updatedAt` 缓存最近内容并渲染基础 Content Viewer。
 */
const Preview: FC = () => {
  const [previewState, setPreviewState] =
    useState<ClipboardPreviewState | null>(null);
  const panelMeasureRef = useRef<HTMLDivElement>(null);
  const { clipboard } = useSnapshot(settingsState);
  const redactSecrets = clipboard.sensitive.redactSecrets;
  const renderState = usePreviewRenderState(previewState);
  const { loadingItemId, payload } = usePreviewPayload(previewState);
  const measuredPanelSize = useMeasuredPanelSize(panelMeasureRef);
  const active = previewState !== null;
  const visibleState = previewState ?? renderState;
  const effectivePanelSize = visibleState
    ? resolveEffectivePanelSize(visibleState.layout, measuredPanelSize, payload)
    : measuredPanelSize;
  const panelRect = visibleState
    ? resolveDynamicPanelRect(visibleState.layout, effectivePanelSize)
    : EMPTY_RECT;
  const panelMeasureStyle = visibleState
    ? resolveMeasurePanelStyle(visibleState.layout)
    : void 0;
  const connector = visibleState
    ? resolveConnector(visibleState.layout.sourceRect, panelRect)
    : EMPTY_CONNECTOR;
  const motionLayout = usePreviewMotion(
    active,
    visibleState?.sessionId ?? null,
    hasMeasuredPanelSize(effectivePanelSize),
    panelRect,
    connector,
  );

  useMount(async () => {
    try {
      const state = await getClipboardPreviewState();
      setPreviewState(state);
    } catch (error) {
      log.error("load preview state failed", error);
    }
  });

  useTauriListen<ClipboardPreviewState | null>(
    TAURI_EVENT.PREVIEW_UPDATED,
    (event) => {
      setPreviewState(event.payload);
    },
  );

  if (!visibleState) {
    return <div className="fixed inset-0 overflow-hidden bg-transparent" />;
  }

  const isLoading = loadingItemId !== null;
  const payloadKey = payload ? cacheKey(payload, redactSecrets) : "empty";
  const { layout } = visibleState;
  const svgStyle = rectStyle(layout.overlayRect);

  return (
    <div className="fixed inset-0 overflow-hidden bg-transparent">
      <motion.svg
        animate={active ? "open" : "closed"}
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 z-10 overflow-visible"
        initial="closed"
        role="presentation"
        style={svgStyle}
        transition={PREVIEW_PANEL_TRANSITION}
        variants={PREVIEW_CONNECTOR_VARIANTS}
        viewBox={`0 0 ${layout.overlayRect.width} ${layout.overlayRect.height}`}
      >
        <motion.path
          className="stroke-ant-primary"
          d={motionLayout.path}
          fill="none"
          strokeLinecap="round"
          strokeWidth="2"
        />
        <motion.circle
          className="fill-ant-container stroke-ant-primary"
          cx={motionLayout.sourceDotX}
          cy={motionLayout.sourceDotY}
          r="4"
          strokeWidth="2.5"
        />
        <motion.circle
          className="fill-ant-container stroke-ant-primary"
          cx={motionLayout.targetDotX}
          cy={motionLayout.targetDotY}
          r="4"
          strokeWidth="2.5"
        />
      </motion.svg>

      <div
        aria-hidden="true"
        className="pointer-events-none invisible absolute top-0 left-0 z-0 flex w-fit min-w-72 max-w-120 flex-col overflow-visible rounded-2 border border-ant-border bg-ant-container/95 shadow-lg backdrop-blur"
        ref={panelMeasureRef}
        style={panelMeasureStyle}
      >
        <PreviewHeader payload={payload} />

        <div className="min-h-0">
          <PreviewContent payload={payload} />
        </div>
      </div>

      <motion.div
        animate={active ? "open" : "closed"}
        className="absolute z-5 flex max-h-120 min-w-72 max-w-120 flex-col overflow-hidden rounded-2 border border-ant-border bg-ant-container/95 shadow-lg backdrop-blur"
        initial="closed"
        style={motionLayout.panelStyle}
        transition={PREVIEW_PANEL_TRANSITION}
        variants={PREVIEW_PANEL_VARIANTS}
      >
        <PreviewContentTransition contentKey={payloadKey}>
          <PreviewHeader payload={payload} />

          <div
            className={cn("min-h-0 flex-1 overflow-auto transition-opacity", {
              "opacity-60": isLoading && payload !== null,
            })}
          >
            <PreviewContent payload={payload} />
          </div>
        </PreviewContentTransition>

        {isLoading && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-ant-mask/10">
            <Spin size="small" />
          </div>
        )}
      </motion.div>
    </div>
  );
};

export default Preview;
