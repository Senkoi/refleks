import { Link } from "react-router-dom";
import { TrainingAdviceWidget } from "@/features/training/TrainingAdviceWidget";
import { useBenchmarkDetailProgress } from "@/features/benchmarks/hooks/useBenchmarkDetailProgress";
import { useBenchmarks } from "@/shared/hooks";
import { benchmarkPath } from "@/shared/lib/navigation";

// Training advice comes exclusively from the workbench service. Benchmark
// progress remains a compact factual summary, with details on its own page.
export function BenchmarkOverviewWidget() {
  const { selectedBenchmark, getBenchmarkByName } = useBenchmarks();
  const benchmark = selectedBenchmark ? getBenchmarkByName(selectedBenchmark) : null;
  const { progress, difficultyIndex } = useBenchmarkDetailProgress(benchmark ?? undefined);
  const rank = progress?.ranks?.[(progress.overallRank ?? 0) - 1]?.name;
  return <div className="min-w-0 space-y-3">
    <TrainingAdviceWidget />
    <div className="flex flex-wrap items-center justify-between gap-2 px-1 text-xs text-surface-muted-foreground">
      <span>基准成绩：{benchmark ? `${benchmark.benchmarkName} · ${benchmark.difficulties?.[difficultyIndex]?.difficultyName ?? ""} · ${rank ?? "尚未定级"}` : "尚未选择基准测试"}</span>
      <Link className="text-primary hover:underline" to={benchmark ? benchmarkPath(benchmark.benchmarkName) : "/benchmarks"}>查看基准测试详情</Link>
    </div>
  </div>;
}
