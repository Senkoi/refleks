import { memo } from "react";
import { CartesianGrid, Line, LineChart, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { PersonalAnchor } from "./api";

type Point = NonNullable<PersonalAnchor["points"]>[number];
const shortDate = (at: number) => new Date(at).toLocaleDateString("zh-CN", { month: "numeric", day: "numeric" });

export default memo(function AnchorTrend({ anchor }: { anchor: PersonalAnchor }) {
  const points = anchor.points ?? [];
  if (!points.length) return <p className="training-muted">暂无可比趋势记录</p>;
  const values = [anchor.medianScore, ...points.map(p => p.score)];
  const low = Math.min(...values), high = Math.max(...values);
  const padding = Math.max((high - low) * .15, high * .05, 1);
  const first = points[0].at, last = points[points.length - 1].at;
  const timeDomain = first === last ? [first - 3600000, last + 3600000] : [first, last];
  return <figure className="training-anchor-trend" aria-label={`${anchor.scenario}成绩趋势：${points.map(p => `${shortDate(p.at)} ${p.score.toFixed(1)}`).join("，")}；当前基准 ${anchor.medianScore.toFixed(1)}`}>
    <ResponsiveContainer width="100%" height={180}>
      <LineChart data={points} margin={{ top: 10, right: 14, left: 0, bottom: 4 }} accessibilityLayer>
        <CartesianGrid vertical={false} stroke="var(--border)" />
        <XAxis dataKey="at" type="number" scale="time" domain={timeDomain} ticks={points.map(p => p.at)} tickFormatter={shortDate} minTickGap={24} tickLine={false} axisLine={false} tick={{ fill: "var(--surface-muted-foreground)", fontSize: 11 }} />
        <YAxis domain={[Math.max(0, low - padding), high + padding]} width={52} tickFormatter={v => Number(v).toFixed(0)} tickLine={false} axisLine={false} tick={{ fill: "var(--surface-muted-foreground)", fontSize: 11 }} />
        <Tooltip content={({ active, payload }) => {
          const point = payload?.[0]?.payload as Point | undefined;
          return active && point ? <div className="training-anchor-tooltip"><span>{new Date(point.at).toLocaleString("zh-CN")}</span><strong>中位成绩 {point.score.toFixed(1)}</strong><span>{point.samples} 局</span></div> : null;
        }} />
        <ReferenceLine y={anchor.medianScore} stroke="var(--surface-muted-foreground)" strokeDasharray="4 4" />
        <Line dataKey="score" name="每次练习的中位成绩" type="linear" stroke="var(--primary)" strokeWidth={2} dot={{ r: 3, fill: "var(--primary)" }} activeDot={{ r: 5 }} isAnimationActive={false} />
      </LineChart>
    </ResponsiveContainer>
    <figcaption>每次练习的中位成绩 · 虚线为当前基准{points.length === 1 && " · 仅一次练习"}</figcaption>
  </figure>;
});
