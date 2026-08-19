const STATUS_LABELS: Record<string, string> = {
  pending: "等待中",
  running: "运行中",
  stopped: "已停止",
  starting: "启动中",
  restarting: "重启中",
  unknown: "未知",
};

export function StatusBadge({ status }: { status: string }) {
  const normalizedStatus = status.toLowerCase();
  const knownStatus = normalizedStatus in STATUS_LABELS ? normalizedStatus : "unknown";
  return <span className={`status status-${knownStatus}`}>{STATUS_LABELS[knownStatus]}</span>;
}
