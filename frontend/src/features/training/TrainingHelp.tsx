import type { ReactNode } from "react";
import { CircleHelp, X } from "lucide-react";
import { Close as PopoverClose } from "@radix-ui/react-popover";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/components/ui/popover";

export default function TrainingHelp({ label, children }: { label: string; children: ReactNode }) {
  return <Popover>
    <PopoverTrigger asChild>
      <button type="button" className="training-help" aria-label={`${label}说明`}>
        <CircleHelp size={15} aria-hidden="true" />
      </button>
    </PopoverTrigger>
    <PopoverContent className="training-help-content" side="bottom" align="start" aria-label={`${label}说明`}>
      <div className="training-help-heading"><strong>{label}</strong><PopoverClose asChild><button type="button" aria-label="关闭说明"><X size={14} aria-hidden="true" /></button></PopoverClose></div>
      <div className="training-help-body">{children}</div>
    </PopoverContent>
  </Popover>;
}
