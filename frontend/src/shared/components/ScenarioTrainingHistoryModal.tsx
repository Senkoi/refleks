import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Modal } from "./Modal";
import {
  getScenarioTrainingHistory,
  groupScenarioHistory,
  type ScenarioHistoryPoint,
} from "../lib/scenarioHistory";

export function ScenarioTrainingHistoryModal({
  scenario,
  onClose,
  supplementary,
  fallback,
}: {
  scenario: string;
  onClose: () => void;
  supplementary?: ReactNode;
  fallback?: ReactNode;
}) {
  const [points, setPoints] = useState<ScenarioHistoryPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selection, setSelection] = useState("");
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    setPoints([]);
    setSelection("");
    getScenarioTrainingHistory(scenario)
      .then((result) => {
        if (!cancelled) setPoints(result);
      })
      .catch((e) => {
        if (!cancelled) setError(String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [scenario, retry]);
  const groups = useMemo(() => groupScenarioHistory(points), [points]);
  const group = groups.find((g) => g.key === selection) ?? groups[0];
  if (!loading && !error && !groups.length && fallback) return <>{fallback}</>;
  return (
    <Modal
      isOpen
      onClose={onClose}
      title={`${scenario} · 训练历史`}
      width="min(880px, 94vw)"
      height="auto"
      className="scenario-history-modal"
      overlayClassName="scenario-history-overlay"
    >
      <div className="scenario-history-body">
        {loading && <p role="status">正在读取训练历史…</p>}
        {!loading && error && (
          <div role="alert">
            <p>{error}</p>
            <button
              type="button"
              className="scenario-history-retry"
              onClick={() => setRetry((v) => v + 1)}
            >
              重新读取
            </button>
          </div>
        )}
        {!loading && !error && !groups.length && (
          <p className="scenario-history-muted">
            还没有这张地图的本地训练记录。
          </p>
        )}
        {!loading && !error && group && (
          <>
            <div className="scenario-history-toolbar">
              <span>
                {points.length} 局记录 · 当前条件 {group.points.length} 局
              </span>
              {groups.length > 1 && (
                <label>
                  记录条件
                  <select
                    aria-label="训练历史记录条件"
                    value={group.key}
                    onChange={(e) => setSelection(e.target.value)}
                  >
                    {groups.map((g) => (
                      <option key={g.key} value={g.key}>
                        {g.label}（{g.points.length} 局）
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>
            <p className="scenario-history-muted">
              {groups.length > 1
                ? "默认显示最近一次使用的条件；不同版本、灵敏度、视野和时长分开查看。"
                : group.label}
            </p>
            <ScenarioRunHistoryChart points={group.points} />
          </>
        )}
        {!loading && supplementary && (
          <section className="scenario-history-supplement">
            <h3>近期参照成绩</h3>
            {supplementary}
          </section>
        )}
      </div>
    </Modal>
  );
}

export function ScenarioRunHistoryChart({
  points,
}: {
  points: ScenarioHistoryPoint[];
}) {
  const scores = points.map((p) => p.score);
  const low = Math.min(...scores),
    high = Math.max(...scores);
  const pad = Math.max((high - low) * 0.15, Math.abs(high) * 0.05, 1);
  const data = points.map((p, index) => ({ ...p, run: index + 1 }));
  const date = (at: number) =>
    new Date(at).toLocaleDateString("zh-CN", {
      month: "numeric",
      day: "numeric",
    });
  return (
    <figure
      className="scenario-history-chart"
      aria-label={`逐局成绩折线图，共 ${points.length} 局`}
    >
      <ResponsiveContainer width="100%" height={280}>
        <LineChart
          data={data}
          margin={{ top: 12, right: 16, left: 0, bottom: 8 }}
          accessibilityLayer
        >
          <CartesianGrid vertical={false} stroke="var(--border)" />
          <XAxis
            dataKey="run"
            tickFormatter={(run) => date(points[Number(run) - 1]?.at)}
            allowDecimals={false}
            minTickGap={28}
            tickLine={false}
            axisLine={false}
            tick={{ fill: "var(--surface-muted-foreground)", fontSize: 11 }}
          />
          <YAxis
            domain={[Math.max(0, low - pad), high + pad]}
            tickFormatter={(v) => Number(v).toFixed(0)}
            width={58}
            tickLine={false}
            axisLine={false}
            tick={{ fill: "var(--surface-muted-foreground)", fontSize: 11 }}
          />
          <Tooltip
            content={({ active, payload }) => {
              const p = payload?.[0]?.payload as
                ScenarioHistoryPoint | undefined;
              return active && p ? (
                <div className="scenario-history-tooltip">
                  <span>{new Date(p.at).toLocaleString("zh-CN")}</span>
                  <strong>成绩 {p.score.toFixed(1)}</strong>
                  <span>
                    命中率{" "}
                    {(p.accuracy <= 1 ? p.accuracy * 100 : p.accuracy).toFixed(
                      1,
                    )}
                    %
                  </span>
                </div>
              ) : null;
            }}
          />
          <Line
            name="成绩"
            dataKey="score"
            type="linear"
            stroke="var(--primary)"
            strokeWidth={2}
            dot={points.length < 100 ? { r: 3, fill: "var(--primary)" } : false}
            activeDot={{ r: 5 }}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
      <figcaption>
        每个点是一局训练成绩，按时间排列
        {points.length === 1 && " · 仅一局记录，暂无趋势"}
      </figcaption>
    </figure>
  );
}
