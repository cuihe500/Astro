import { Plus, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { EmptyState, ErrorState } from "../../../components/Feedback";
import { errorMessage } from "../../../lib/api";
import { getApps } from "../api";
import { StatusBadge } from "../components/StatusBadge";
import type { App } from "../types";

function formatDate(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "-" : date.toLocaleString("zh-CN", { hour12: false });
}

export function AppsListPage() {
  const location = useLocation();
  const navigationMessage = (location.state as { message?: string } | null)?.message;
  const [apps, setApps] = useState<App[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const loadApps = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setApps(await getApps());
    } catch (requestError) {
      setError(errorMessage(requestError, "应用列表加载失败。"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // 首次进入页面时同步远端数据，状态更新来自请求结果。
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadApps();
  }, [loadApps]);

  return (
    <section className="page" aria-labelledby="apps-title">
      <header className="page-header">
        <div>
          <p className="eyebrow">工作台</p>
          <h1 id="apps-title">我的应用</h1>
        </div>
        <div className="page-actions">
          <button
            className="icon-button"
            type="button"
            onClick={() => void loadApps()}
            disabled={loading}
            aria-label="刷新应用列表"
            title="刷新应用列表"
          >
            <RefreshCw className={loading ? "spin" : undefined} size={19} aria-hidden="true" />
          </button>
          <Link className="button button-primary" to="/apps/new">
            <Plus size={18} aria-hidden="true" />
            创建应用
          </Link>
        </div>
      </header>

      {navigationMessage ? <div className="feedback feedback-success" role="status">{navigationMessage}</div> : null}

      {error && apps.length === 0 ? <ErrorState message={error} onRetry={() => void loadApps()} /> : null}
      {loading && apps.length === 0 ? <div className="list-loading" role="status">正在加载应用...</div> : null}
      {!loading && !error && apps.length === 0 ? (
        <EmptyState title="还没有应用">
          <Link className="button button-primary" to="/apps/new">
            <Plus size={18} aria-hidden="true" />
            创建第一个应用
          </Link>
        </EmptyState>
      ) : null}

      {apps.length > 0 ? (
        <div className="app-list-wrap">
          {error ? <div className="feedback feedback-error" role="alert">{error}</div> : null}
          <div className="app-list-header" aria-hidden="true">
            <span>名称</span><span>镜像</span><span>状态</span><span>副本</span><span>更新时间</span>
          </div>
          <ul className="app-list">
            {apps.map((app) => (
              <li key={app.id}>
                <Link className="app-row" to={`/apps/${app.id}`}>
                  <strong>{app.name}</strong>
                  <span className="app-image" title={app.image}>{app.image}</span>
                  <span><StatusBadge status={app.status} /></span>
                  <span><span className="mobile-label">副本 </span>{app.replicas}</span>
                  <time dateTime={app.updated_at}>{formatDate(app.updated_at)}</time>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
