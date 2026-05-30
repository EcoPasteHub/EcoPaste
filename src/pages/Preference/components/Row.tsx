import type { ReactNode } from "react";

interface RowProps {
  label: ReactNode;
  description?: ReactNode;
  control: ReactNode;
}

const Row = ({ label, description, control }: RowProps) => (
  <div className="flex items-start justify-between gap-4 py-2.5">
    <div className="min-w-0 flex-1">
      <div className="text-sm">{label}</div>
      {description && (
        <div className="mt-0.5 text-default-500 text-xs">{description}</div>
      )}
    </div>
    <div className="shrink-0">{control}</div>
  </div>
);

export default Row;
