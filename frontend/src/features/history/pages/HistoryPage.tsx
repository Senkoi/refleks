import { useEffect, useRef, useState } from "react";
import { PerformanceVsSensWidget } from "../components/PerformanceVsSensWidget";
import { SessionScenarioRadarWidget } from "../components/SessionScenarioRadarWidget";
import "../history.css";
import { trainingStatusLabel } from "@/features/training/progress";
import { Loading } from "@/shared/components";
import { useStore } from "@/shared/hooks";
import { useI18n } from "@/shared/lib";
import { HistoryRunDetailPane } from "../components/HistoryRunDetailPane";
import { HistoryRunList } from "../components/HistoryRunList";
import { HistorySessionList } from "../components/HistorySessionList";
import { HistorySessionOverview } from "../components/HistorySessionOverview";
import { useHistoryPageState } from "../hooks/useHistoryPageState";

export function HistoryPage() {
  const { t } = useI18n();
  const root = useRef<HTMLDivElement>(null);
  const [compact, setCompact] = useState(false);
  const [pane, setPane] = useState<"sessions" | "overview" | "runs">(
    "overview",
  );
  const [advancedView, setAdvancedView] = useState<"" | "sens" | "radar">("");
  const allSessions = useStore((s) => s.sessions);
  const sessionGrouping = useStore((s) => s.sessionGrouping);
  const sessionError = useStore((s) => s.sessionError);
  const runHydration = useStore((s) => s.runHydration);
  const {
    planId,
    setPlanId,
    plans,
    planLoading,
    planError,
    sessions,
    filteredSessions,
    selectedSession,
    selectedSessionId,
    setSelectedSessionId,
    sessionQuery,
    setSessionQuery,
    sessionListCollapsed,
    setSessionListCollapsed,
    sessionSort,
    setSessionSort,
    sessionFilterPb,
    setSessionFilterPb,
    runQuery,
    setRunQuery,
    sessionRuns,
    filteredSessionRuns,
    runInspectorOpen,
    setRunInspectorOpen,
    runListCollapsed,
    setRunListCollapsed,
    runSort,
    setRunSort,
    runFilterPb,
    setRunFilterPb,
    inspectorTab,
    setInspectorTab,
    selectedScenario,
    setSelectedScenario,
    primaryRun,
    compareRun,
    pbRunForPrimary,
    globalPbByScenario,
    selectRun,
    compareRunWithPrimary,
    comparePb,
    clearPrimaryRun,
    clearComparison,
  } = useHistoryPageState();

  useEffect(() => {
    const element = root.current;
    if (!element) return;
    const observer = new ResizeObserver((entries) =>
      setCompact(entries[0].contentRect.width < 1100),
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [runHydration.loading, sessionGrouping]);
  if (allSessions.length === 0 && (runHydration.loading || sessionGrouping)) {
    const label =
      runHydration.total > 0
        ? t("history.page.loadingProgress", {
            loaded: Math.min(runHydration.loaded, runHydration.total),
            total: runHydration.total,
          })
        : t("history.page.loading");

    return <Loading label={label} />;
  }

  return (
    <div
      ref={root}
      className="history-page flex flex-1 flex-col overflow-hidden text-sm"
    >
      <div className="flex flex-wrap items-center gap-3 border-b px-5 py-3">
        <details>
          <summary className="cursor-pointer text-xs">
            {t("history.page.planFilter")}
          </summary>
          <label className="mt-2 flex flex-wrap items-center gap-2 text-xs">
            <select
              aria-label={t("history.page.planFilter")}
              value={planId}
              onChange={(e) => setPlanId(e.target.value)}
              className="max-w-full rounded border bg-background p-1"
            >
              <option value="">{t("history.page.allPlans")}</option>
              {planId && !plans.some((p) => p.id === planId) && (
                <option value={planId}>{t("history.page.selectedPlan")}</option>
              )}
              {plans.map((p) => (
                <option key={p.id} value={p.id}>
                  {new Date(p.created).toLocaleString()} · {p.minutes} 分钟 ·{" "}
                  {trainingStatusLabel(p)}
                </option>
              ))}
            </select>
          </label>
        </details>
        {planId && (
          <button
            className="rounded border px-2 py-1 text-xs text-primary"
            onClick={() => setPlanId("")}
          >
            {t("history.page.clearPlanFilter")}
          </button>
        )}
        {planId && (
          <span
            role={planError ? "alert" : "status"}
            className="text-xs text-surface-muted-foreground"
          >
            {planLoading
              ? t("history.page.planLoading")
              : planError
                ? t("history.page.planError")
                : sessions.length
                  ? t("history.page.planFiltered")
                  : t("history.page.planEmpty")}
          </span>
        )}
        <details>
          <summary className="cursor-pointer text-xs">
            {t("history.page.optionalAnalysis")}
          </summary>
          <div className="flex flex-wrap gap-3 py-2">
            <button
              onClick={() => {
                setAdvancedView("");
                setPane("overview");
              }}
            >
              {t("history.page.sessionOverview")}
            </button>
            <button
              onClick={() => {
                setAdvancedView("sens");
                setPane("overview");
              }}
            >
              {t("history.page.sensitivityChart")}
            </button>
            <button
              onClick={() => {
                setAdvancedView("radar");
                setPane("overview");
              }}
            >
              {t("history.page.scenarioChart")}
            </button>
          </div>
        </details>
        {compact && (
          <nav
            aria-label={t("history.page.paneNavigation")}
            className="flex flex-wrap gap-2"
          >
            {(["sessions", "overview", "runs"] as const).map((value) => (
              <button
                key={value}
                aria-pressed={pane === value}
                className={`rounded border px-2 py-1 text-xs ${pane === value ? "text-primary" : "text-surface-muted-foreground"}`}
                onClick={() => setPane(value)}
              >
                {t(
                  value === "sessions"
                    ? "history.page.sessionsPane"
                    : value === "runs"
                      ? "history.page.runsPane"
                      : "history.page.detailPane",
                )}
              </button>
            ))}
          </nav>
        )}
        {sessionError && (
          <span role="alert" className="text-xs text-destructive">
            {sessionError}
          </span>
        )}
      </div>
      <div className="flex min-h-0 flex-1 gap-4 overflow-hidden p-4 xl:p-5">
        {(!compact || pane === "sessions") && (
          <HistorySessionList
            sessions={filteredSessions}
            selectedSessionId={selectedSessionId}
            collapsed={!compact && sessionListCollapsed}
            query={sessionQuery}
            onQueryChange={setSessionQuery}
            onSelectSession={(id) => {
              setSelectedSessionId(id);
              setPane("overview");
              setAdvancedView("");
            }}
            onToggleCollapsed={() =>
              compact ? setPane("overview") : setSessionListCollapsed((v) => !v)
            }
            sort={sessionSort}
            onSortChange={setSessionSort}
            filterPb={sessionFilterPb}
            onFilterPbChange={setSessionFilterPb}
            compact={compact}
          />
        )}

        {/* Main content area: session overview or inspector */}
        {(!compact || pane === "overview") && (
          <div className="min-h-0 min-w-0 flex-1">
            {advancedView === "sens" ? (
              <PerformanceVsSensWidget allowScopeSelection className="h-full" />
            ) : advancedView === "radar" ? (
              <SessionScenarioRadarWidget className="h-full" />
            ) : runInspectorOpen ? (
              <HistoryRunDetailPane
                primaryRun={primaryRun}
                compareRun={compareRun}
                activeTab={inspectorTab}
                onTabChange={setInspectorTab}
                onClose={() => setRunInspectorOpen(false)}
                onClearPrimaryRun={clearPrimaryRun}
                onClearComparison={clearComparison}
                isPrimaryPb={
                  !!primaryRun &&
                  !!pbRunForPrimary &&
                  primaryRun.id === pbRunForPrimary.id
                }
                isComparePb={
                  !!compareRun &&
                  !!pbRunForPrimary &&
                  compareRun.id === pbRunForPrimary.id
                }
                onComparePb={comparePb}
              />
            ) : (
              <HistorySessionOverview
                session={selectedSession}
                sessions={sessions}
                sessionRuns={sessionRuns}
                selectedScenario={selectedScenario}
                onSelectScenario={setSelectedScenario}
                onSelectRun={(id) => {
                  selectRun(id);
                  setPane("overview");
                }}
                globalPbByScenario={globalPbByScenario}
              />
            )}
          </div>
        )}

        {(!compact || pane === "runs") && (
          <HistoryRunList
            session={selectedSession}
            runs={filteredSessionRuns}
            query={runQuery}
            primaryRun={primaryRun}
            compareRun={compareRun}
            collapsed={!compact && runListCollapsed}
            inspectorOpen={runInspectorOpen}
            selectedScenario={selectedScenario}
            onQueryChange={setRunQuery}
            onToggleCollapsed={() =>
              compact ? setPane("overview") : setRunListCollapsed((v) => !v)
            }
            onToggleInspector={() => {
              setRunInspectorOpen((v) => !v);
              setPane("overview");
              setAdvancedView("");
            }}
            onSelectRun={(id) => {
              selectRun(id);
              setPane("overview");
              setAdvancedView("");
            }}
            onCompareRun={(id) => {
              compareRunWithPrimary(id);
              setPane("overview");
              setAdvancedView("");
            }}
            sort={runSort}
            onSortChange={setRunSort}
            filterPb={runFilterPb}
            onFilterPbChange={setRunFilterPb}
            compact={compact}
          />
        )}
      </div>
    </div>
  );
}
