import { Loading } from "@/shared/components";
import { useI18n } from "@/shared/lib";
import { useStore } from "@/shared/hooks";
import { BenchmarkOverviewWidget } from "../components/BenchmarkOverviewWidget";
import {
  LastRunWidget,
  RecentScoresWidget,
  SessionTimeWidget,
  StreakPlaytimeWidget,
} from "../components/SessionWidgets";
import { useRecentSessionSnapshot } from "../hooks/useRecentSessionSnapshot";

export function OverviewPage() {
  const { t } = useI18n();
  const snapshot = useRecentSessionSnapshot();
  const sessions = useStore((s) => s.sessions);
  const sessionGrouping = useStore((s) => s.sessionGrouping);
  const sessionError = useStore((s) => s.sessionError);
  const runHydration = useStore((s) => s.runHydration);

  if (sessions.length === 0 && (runHydration.loading || sessionGrouping)) {
    const label =
      runHydration.total > 0
        ? t("overview.page.loadingHistoryProgress", {
            loaded: Math.min(runHydration.loaded, runHydration.total),
            total: runHydration.total,
          })
        : t("overview.page.loadingHistory");

    return <Loading label={label} />;
  }

  return (
    <div className="flex-1 overflow-auto text-sm">
      <div className="grid min-w-0 gap-4 p-5">
        {sessionError && (
          <p role="alert" className="text-destructive">
            {sessionError}
          </p>
        )}
        <BenchmarkOverviewWidget />
        <div className="grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="min-w-0 space-y-4">
            <LastRunWidget snapshot={snapshot} />
            <RecentScoresWidget snapshot={snapshot} />
          </div>
          <div className="grid min-w-0 grid-cols-1 content-start gap-4 sm:grid-cols-2">
            <SessionTimeWidget snapshot={snapshot} />
            <StreakPlaytimeWidget snapshot={snapshot} />
          </div>
        </div>
      </div>
    </div>
  );
}
