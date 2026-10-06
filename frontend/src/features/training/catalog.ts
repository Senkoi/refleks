import type { Scenario } from "./api";

export type FileFilter = "all" | "verified" | "parsed" | "issues" | "missing";
export type DifficultyFilter =
  "all" | "evidence" | "precision" | "benchmark" | "calibrated" | "unfitted";
export const fileLabels: Record<string, string> = {
  verified: "已下载 · 配置验证通过",
  parsed: "已解析 · 配置待验证",
  pending: "等待文件稳定",
  invalid: "文件解析失败",
  conflict: "同名版本冲突",
  missing: "本地文件不可用",
};
export const evidenceLabels: Record<string, string> = {
  precision_reference: "同家族精度对照",
  benchmark_reference: "原生 benchmark 参考",
  calibrated: "总难度已标定",
  unfitted: "尚无难度评估依据",
};
export const skillLabels: Record<string, string> = {
  static: "静态点击",
  dynamic: "动态点击",
  smooth: "精确追踪",
  reactive: "反应追踪",
  switching: "目标切换",
  unknown: "待分类",
};
export type CatalogFilters = {
  skill?: string;
  experience?: "all" | "played" | "unplayed";
  feedback?: "all" | "liked" | "disliked" | "excluded";
};

export function filterCatalog(
  rows: Scenario[],
  search: string,
  file: FileFilter,
  difficulty: DifficultyFilter,
  filters: CatalogFilters = {},
  played: ReadonlySet<string> = new Set(),
): Scenario[] {
  const query = search.trim().toLowerCase();
  return rows.filter((s) => {
    if (
      !`${s.name} ${s.skill} ${skillLabels[s.skill] ?? ""} ${s.sources?.map((x) => x.title).join(" ")}`
        .toLowerCase()
        .includes(query)
    )
      return false;
    if (filters.skill && s.skill !== filters.skill) return false;
    const hasPlayed = played.has(s.name.trim().toLowerCase());
    if (filters.experience === "played" && !hasPlayed) return false;
    if (filters.experience === "unplayed" && hasPlayed) return false;
    if (filters.feedback === "excluded" && s.enabled) return false;
    if (
      ["liked", "disliked"].includes(filters.feedback ?? "") &&
      s.preference !== filters.feedback
    )
      return false;
    const e = s.evaluation;
    const status = e?.fileStatus ?? "missing";
    if (file === "verified" && status !== "verified") return false;
    if (file === "parsed" && !["verified", "parsed"].includes(status))
      return false;
    if (
      file === "issues" &&
      !["parsed", "pending", "invalid", "conflict"].includes(status)
    )
      return false;
    if (file === "missing" && status !== "missing") return false;
    if (difficulty === "evidence" && !e?.hasDifficultyEvidence) return false;
    if (difficulty === "precision" && !e?.hasPrecisionReference) return false;
    if (difficulty === "benchmark" && !e?.hasBenchmarkReference) return false;
    if (difficulty === "calibrated" && e?.difficultyStatus !== "calibrated")
      return false;
    if (difficulty === "unfitted" && e?.hasDifficultyEvidence) return false;
    return true;
  });
}

export function catalogPage(rows: Scenario[], requested: number) {
  const pages = Math.max(1, Math.ceil(rows.length / 100));
  const page = Math.max(0, Math.min(requested, pages - 1));
  return { page, pages, rows: rows.slice(page * 100, (page + 1) * 100) };
}
