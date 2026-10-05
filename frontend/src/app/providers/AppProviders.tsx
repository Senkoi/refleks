import { I18nProvider } from "@/shared/lib/i18n";
import { BenchmarkProvider, StoreProvider } from "@/shared/hooks";
import type { ReactNode } from "react";
import { ScenarioHistoryProvider } from "@/shared/components/ScenarioHistoryLink";

import { TrainingProgressProvider } from "@/features/training/TrainingProgressProvider";

interface AppProvidersProps {
  children: ReactNode;
}

export function AppProviders({ children }: AppProvidersProps) {
  return (
    <I18nProvider>
      <StoreProvider>
        <TrainingProgressProvider><BenchmarkProvider><ScenarioHistoryProvider>{children}</ScenarioHistoryProvider></BenchmarkProvider></TrainingProgressProvider>
      </StoreProvider>
    </I18nProvider>
  );
}
