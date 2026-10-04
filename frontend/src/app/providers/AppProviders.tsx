import { I18nProvider } from "@/shared/lib/i18n";
import { BenchmarkProvider, StoreProvider } from "@/shared/hooks";
import type { ReactNode } from "react";
import { ScenarioHistoryProvider } from "@/shared/components/ScenarioHistoryLink";

interface AppProvidersProps {
  children: ReactNode;
}

export function AppProviders({ children }: AppProvidersProps) {
  return (
    <I18nProvider>
      <StoreProvider>
        <BenchmarkProvider><ScenarioHistoryProvider>{children}</ScenarioHistoryProvider></BenchmarkProvider>
      </StoreProvider>
    </I18nProvider>
  );
}
