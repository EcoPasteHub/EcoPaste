import { useMount, useUnmount } from "ahooks";
import { Checkbox, Empty } from "antd";
import type { FC } from "react";
import { useEffect, useEffectEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  detectLegacyData,
  importLegacyData,
  type LegacyImportSelection,
  type OnboardingLegacyDataDetection,
} from "@/commands";
import CustomIconButton from "@/components/CustomIconButton";
import { log } from "@/utils/log";
import type { OnboardingStepProps } from "../types";
import OnboardingStepLayout from "./OnboardingStepLayout";

const LegacyImportStep: FC<OnboardingStepProps> = (props) => {
  const { onActionLoadingChange, onActionsChange, requestNext } = props;
  const { t } = useTranslation("onboarding");
  const [detection, setDetection] =
    useState<OnboardingLegacyDataDetection | null>(null);
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [selectedTypes, setSelectedTypes] = useState<LegacyImportSelection[]>([
    "normal",
    "favorite",
  ]);

  useEffect(() => {
    onActionLoadingChange?.(loading || importing);
  }, [importing, loading, onActionLoadingChange]);

  useUnmount(() => {
    onActionLoadingChange?.(false);
    onActionsChange?.(null);
  });

  const runDetection = async () => {
    setLoading(true);

    try {
      const result = await detectLegacyData();
      setDetection(result);
    } catch (error) {
      log.warn("detect legacy data failed", error);
    } finally {
      setLoading(false);
    }
  };

  useMount(() => {
    void runDetection();
  });

  const handleRefreshClick = async () => {
    await runDetection();
  };

  const handleImportSelectionChange = (
    nextChecked: LegacyImportSelection[],
  ) => {
    setSelectedTypes(nextChecked);
  };

  const handleImportClick = async () => {
    if (selectedTypes.length === 0) {
      return false;
    }

    setImporting(true);

    try {
      await importLegacyData(selectedTypes);
      const refreshed = await detectLegacyData();
      setDetection(refreshed);
      return true;
    } catch (error) {
      log.warn("import legacy data failed", error);
      return false;
    } finally {
      setImporting(false);
    }
  };

  const handleImportAndContinue = useEffectEvent(async () => {
    const imported = await handleImportClick();
    if (!imported) {
      return;
    }

    await requestNext?.();
  });

  const handleSkip = useEffectEvent(async () => {
    await requestNext?.();
  });

  const hasImportableData =
    detection?.importableItemCount !== void 0 &&
    detection.importableItemCount > 0;

  useEffect(() => {
    onActionsChange?.({
      extraText: t("actions.skip"),
      nextDisabled: !hasImportableData || selectedTypes.length === 0,
      nextText: t("legacyImport.importAction"),
      onExtra: handleSkip,
      onNext: handleImportAndContinue,
    });
  }, [hasImportableData, onActionsChange, selectedTypes.length, t]);

  return (
    <OnboardingStepLayout
      description={t("legacyImport.description")}
      icon={<i aria-hidden="true" className="i-lucide:database-backup" />}
      title={t("legacyImport.title")}
    >
      <section className="flex flex-col gap-3 rounded-2 border border-ant-border-secondary bg-ant-container p-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h2 className="m-0 font-semibold text-ant-text text-lg">
              {t("legacyImport.cardTitle")}
            </h2>
            <p className="m-0 mt-1 text-ant-secondary text-sm">
              {t("legacyImport.cardDescription")}
            </p>
          </div>
          <CustomIconButton
            icon={<i aria-hidden="true" className="i-lucide:refresh-cw" />}
            loading={loading}
            onClick={handleRefreshClick}
          >
            {t("legacyImport.refresh")}
          </CustomIconButton>
        </div>

        {detection?.found ? (
          <div className="rounded-2 border border-ant-border-secondary bg-ant-bg-layout/30 px-4 py-3 text-ant-secondary text-sm">
            <span className="font-medium text-ant-text">
              {t("legacyImport.summary.detectedValue")}
            </span>
            <span className="mx-2">·</span>
            <span>
              {t("legacyImport.summary.compactItems", {
                count: detection.importableItemCount,
              })}
            </span>
            <span className="mx-2">·</span>
            <span>
              {t("legacyImport.summary.compactSize", {
                size: formatBytes(detection.totalBytes),
              })}
            </span>
          </div>
        ) : (
          <Empty
            description={
              loading
                ? t("legacyImport.detecting")
                : t("legacyImport.emptyDescription")
            }
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        )}

        {hasImportableData && detection ? (
          <section className="flex flex-col gap-2 rounded-2 border border-ant-border-secondary/70 bg-ant-bg-layout/30 p-3">
            <span className="font-medium text-ant-text text-sm">
              {t("legacyImport.importSectionTitle")}
            </span>
            <Checkbox.Group
              className="flex flex-col gap-2"
              onChange={(checked) => {
                handleImportSelectionChange(checked as LegacyImportSelection[]);
              }}
              value={selectedTypes}
            >
              <div className="flex items-center justify-between rounded-2 border border-ant-border-secondary bg-ant-container px-3 py-2.5">
                <Checkbox value="normal">
                  {t("legacyImport.types.normal")}
                </Checkbox>
                <span className="font-semibold text-ant-text text-sm">
                  {detection.normalItemCount}
                </span>
              </div>
              <div className="flex items-center justify-between rounded-2 border border-ant-border-secondary bg-ant-container px-3 py-2.5">
                <Checkbox value="favorite">
                  {t("legacyImport.types.favorite")}
                </Checkbox>
                <span className="font-semibold text-ant-text text-sm">
                  {detection.favoriteItemCount}
                </span>
              </div>
            </Checkbox.Group>
          </section>
        ) : null}
      </section>
    </OnboardingStepLayout>
  );
};

export default LegacyImportStep;

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;

  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[unitIndex]}`;
}
