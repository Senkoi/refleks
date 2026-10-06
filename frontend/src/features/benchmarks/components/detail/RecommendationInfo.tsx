import { InfoTooltip } from "@/shared/components";
import { useI18n } from "@/shared/lib/i18n";

export function RecommendationInfo() {
  const { t } = useI18n();
  return (
    <InfoTooltip
      side="bottom"
      className="max-w-72"
      ariaLabel={t("benchmarks.recommendationInfo.ariaLabel")}
    >
      <div className="space-y-2 text-xs">
        <p>{t("benchmarks.recommendationInfo.description")}</p>
        <p>{t("benchmarks.scenarioRow.notInPreview")}</p>
      </div>
    </InfoTooltip>
  );
}
