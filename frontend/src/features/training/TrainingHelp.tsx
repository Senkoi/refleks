import type { ReactNode } from "react";
import { CircleHelp } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/components/ui/popover";

export default function TrainingHelp({ label, children }: { label: string; children: ReactNode }) {
  return <Popover>
    <PopoverTrigger asChild>
      <button type="button" className="training-help" aria-label={`${label}说明`}>
        <CircleHelp size={15} aria-hidden="true" />
      </button>
    </PopoverTrigger>
    <PopoverContent className="training-help-content" side="bottom" align="start" aria-label={`${label}说明`}>
      {children}
    </PopoverContent>
  </Popover>;
}
