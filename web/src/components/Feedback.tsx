import { Inbox, LoaderCircle, RefreshCw } from "lucide-react";
import type { ReactNode } from "react";

export function LoadingState({ label = "正在加载" }: { label?: string }) {
  return (
    <div className="page-state" role="status" aria-live="polite">
      <LoaderCircle className="spin" aria-hidden="true" />
      <p>{label}</p>
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="page-state page-state-error" role="alert">
      <p>{message}</p>
      {onRetry ? (
        <button className="button button-secondary" type="button" onClick={onRetry}>
          <RefreshCw size={17} aria-hidden="true" />
          重试
        </button>
      ) : null}
    </div>
  );
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="page-state page-state-empty">
      <Inbox aria-hidden="true" />
      <h2>{title}</h2>
      {children}
    </div>
  );
}
