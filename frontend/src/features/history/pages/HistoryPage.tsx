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
  const allSessions = useStore((s) => s.sessions);
  const sessionGrouping=useStore(s=>s.sessionGrouping);
  const sessionError=useStore(s=>s.sessionError);
  const runHydration = useStore((s) => s.runHydration);
  const {
    planId,setPlanId,plans,planLoading,planError,
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
    <div className="flex flex-1 flex-col overflow-hidden text-sm">
      <div className="flex flex-wrap items-center gap-3 border-b px-5 py-3">
        <label className="flex items-center gap-2 text-xs">训练范围<select aria-label="按训练计划筛选对局" value={planId} onChange={e=>setPlanId(e.target.value)} className="max-w-96 rounded border bg-background p-1"><option value="">全部训练时段</option>{planId&&!plans.some(p=>p.id===planId)&&<option value={planId}>所选训练计划</option>}{plans.map(p=><option key={p.id} value={p.id}>{new Date(p.created).toLocaleString()} · {p.minutes} 分钟 · {trainingStatusLabel(p)}</option>)}</select></label>
        {planId&&<span className="text-xs text-surface-muted-foreground">{planLoading?"正在读取本计划对局…":planError?"计划对局暂时无法读取":sessions.length?"仅显示本计划的对局，按所属训练时段排列":"暂无已关联对局；生成列表不代表完成训练"}</span>}
        {sessionError&&<span role="alert" className="text-xs text-destructive">{sessionError}</span>}
      </div>
      <div className="flex min-h-0 flex-1 gap-4 overflow-hidden p-4 xl:p-5">
        <HistorySessionList
          sessions={filteredSessions}
          selectedSessionId={selectedSessionId}
          collapsed={sessionListCollapsed}
          query={sessionQuery}
          onQueryChange={setSessionQuery}
          onSelectSession={setSelectedSessionId}
          onToggleCollapsed={() => setSessionListCollapsed((v) => !v)}
          sort={sessionSort}
          onSortChange={setSessionSort}
          filterPb={sessionFilterPb}
          onFilterPbChange={setSessionFilterPb}
        />

        {/* Main content area: session overview or inspector */}
        <div className="min-h-0 min-w-0 flex-1">
          {runInspectorOpen ? (
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
              onSelectRun={selectRun}
              globalPbByScenario={globalPbByScenario}
            />
          )}
        </div>

        <HistoryRunList
          session={selectedSession}
          runs={filteredSessionRuns}
          query={runQuery}
          primaryRun={primaryRun}
          compareRun={compareRun}
          collapsed={runListCollapsed}
          inspectorOpen={runInspectorOpen}
          selectedScenario={selectedScenario}
          onQueryChange={setRunQuery}
          onToggleCollapsed={() => setRunListCollapsed((v) => !v)}
          onToggleInspector={() => setRunInspectorOpen((v) => !v)}
          onSelectRun={selectRun}
          onCompareRun={compareRunWithPrimary}
          sort={runSort}
          onSortChange={setRunSort}
          filterPb={runFilterPb}
          onFilterPbChange={setRunFilterPb}
        />
      </div>
    </div>
  );
}
