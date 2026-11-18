import { errorStatusLabel } from "../api/errors";
import { Button } from "./Button";
import { Icon } from "./Icon";
import { StatePanel } from "./StatePanel";

interface ErrorStateProps {
  title: string;
  description: string;
  error: unknown;
  onRetry: () => void;
  retryLabel?: string;
}

export function ErrorState({
  title,
  description,
  error,
  onRetry,
  retryLabel = "Try again",
}: ErrorStateProps) {
  const status = errorStatusLabel(error);
  return (
    <StatePanel
      title={title}
      description={description}
      visual={
        <div className="flex size-14 items-center justify-center rounded-full bg-danger-soft text-danger">
          <Icon name="alert" size={28} strokeWidth={1.8} />
        </div>
      }
    >
      <Button onClick={onRetry} icon={<Icon name="retry" size={18} />}>
        {retryLabel}
      </Button>
      {status && <div className="mt-1 text-xs text-text-3">{status}</div>}
    </StatePanel>
  );
}
