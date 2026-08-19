import { ArrowLeft, CircleCheck, Play, RefreshCw, RotateCcw, Square, Trash2 } from "lucide-react";
import { type FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { EmptyState, ErrorState, LoadingState } from "../../../components/Feedback";
import { errorMessage } from "../../../lib/api";
import { deleteApp, getApp, getAppLogs, runLifecycleAction } from "../api";
import { StatusBadge } from "../components/StatusBadge";
import type { App, LifecycleAction } from "../types";

function formatDate(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "-" : date.toLocaleString("zh-CN", { hour12: false });
}

const ACTION_LABELS: Record<LifecycleAction, string> = {
  start: "启动",
  stop: "停止",
  restart: "重启",
};

export function AppDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const deleteDialog = useRef<HTMLDialogElement>(null);
  const [app, setApp] = useState<App | null>(null);
  const [loading, setLoading] = useState(true);
  const [appError, setAppError] = useState("");
  const [logs, setLogs] = useState("");
  const [logsLoading, setLogsLoading] = useState(true);
  const [logsError, setLogsError] = useState("");
  const [pendingAction, setPendingAction] = useState<LifecycleAction | "delete" | null>(null);
  const [feedback, setFeedback] = useState<{ message: string; error: boolean } | null>(() => {
    const message = (location.state as { message?: string } | null)?.message;
    return message ? { message, error: false } : null;
  });

  const loadApp = useCallback(async () => {
    setLoading(true);
    setAppError("");
    try {
      setApp(await getApp(id));
    } catch (requestError) {
      setAppError(errorMessage(requestError, "应用详情加载失败。"));
    } finally {
      setLoading(false);
    }
  }, [id]);

  const loadLogs = useCallback(async () => {
    setLogsLoading(true);
    setLogsError("");
    try {
      setLogs(await getAppLogs(id));
    } catch (requestError) {
      setLogsError(errorMessage(requestError, "日志加载失败。"));
    } finally {
      setLogsLoading(false);
    }
  }, [id]);

  useEffect(() => {
    // 首次进入页面时同步详情和日志，状态更新来自请求结果。
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadApp();
    void loadLogs();
  }, [loadApp, loadLogs]);

  async function runAction(action: LifecycleAction) {
    if (!app || pendingAction) return;
    setPendingAction(action);
    setFeedback(null);
    try {
      await runLifecycleAction(app.id, action);
      setFeedback({ message: `${ACTION_LABELS[action]}请求已提交。`, error: false });
      await loadApp();
    } catch (requestError) {
      setFeedback({ message: errorMessage(requestError, `${ACTION_LABELS[action]}失败，请重试。`), error: true });
    } finally {
      setPendingAction(null);
    }
  }

  function openDeleteDialog() {
    deleteDialog.current?.showModal();
  }

  async function confirmDelete(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!app || pendingAction) return;
    setPendingAction("delete");
    setFeedback(null);
    try {
      await deleteApp(app.id);
      deleteDialog.current?.close();
      navigate("/apps", { replace: true, state: { message: "应用已删除。" } });
    } catch (requestError) {
      setFeedback({ message: errorMessage(requestError, "删除应用失败，请重试。"), error: true });
      setPendingAction(null);
    }
  }

  if (loading) return <section className="page"><LoadingState label="正在加载应用详情" /></section>;
  if (appError || !app) {
    return (
      <section className="page" aria-labelledby="detail-error-title">
        <Link className="back-link" to="/apps"><ArrowLeft size={17} aria-hidden="true" />返回应用列表</Link>
        <div className="sr-only" id="detail-error-title">应用详情</div>
        <ErrorState message={appError || "应用不存在。"} onRetry={() => void loadApp()} />
      </section>
    );
  }

  const status = app.status.toLowerCase();
  const hasLifecycleActions = status === "running" || status === "stopped";

  return (
    <section className="page" aria-labelledby="detail-title">
      <Link className="back-link" to="/apps"><ArrowLeft size={17} aria-hidden="true" />返回应用列表</Link>
      <header className="page-header detail-header">
        <div>
          <p className="eyebrow">应用详情</p>
          <h1 id="detail-title">{app.name}</h1>
        </div>
        <div className="page-actions">
          <button
            className="icon-button"
            type="button"
            onClick={() => { void loadApp(); void loadLogs(); }}
            disabled={pendingAction !== null}
            aria-label="刷新应用详情和日志"
            title="刷新应用详情和日志"
          >
            <RefreshCw size={19} aria-hidden="true" />
          </button>
          <button className="button button-danger" type="button" onClick={openDeleteDialog} disabled={pendingAction !== null}>
            <Trash2 size={17} aria-hidden="true" />
            删除应用
          </button>
        </div>
      </header>

      {feedback ? (
        <div className={`feedback ${feedback.error ? "feedback-error" : "feedback-success"}`} role={feedback.error ? "alert" : "status"}>
          {!feedback.error ? <CircleCheck size={17} aria-hidden="true" /> : null}
          {feedback.message}
        </div>
      ) : null}

      <div className="detail-layout">
        <section className="detail-section" aria-labelledby="info-title">
          <div className="section-heading"><h2 id="info-title">基础信息</h2><StatusBadge status={app.status} /></div>
          <dl className="detail-grid">
            <div><dt>应用名称</dt><dd>{app.name}</dd></div>
            <div><dt>容器镜像</dt><dd className="breakable">{app.image}</dd></div>
            <div><dt>副本数</dt><dd>{app.replicas}</dd></div>
            <div><dt>创建时间</dt><dd>{formatDate(app.created_at)}</dd></div>
            <div><dt>更新时间</dt><dd>{formatDate(app.updated_at)}</dd></div>
          </dl>
        </section>

        <section className="detail-section" aria-labelledby="actions-title">
          <div className="section-heading"><h2 id="actions-title">应用操作</h2></div>
          <div className="lifecycle-actions">
            {status === "stopped" ? (
              <button className="button button-primary" type="button" onClick={() => void runAction("start")} disabled={pendingAction !== null}>
                <Play size={17} aria-hidden="true" />{pendingAction === "start" ? "正在启动" : "启动"}
              </button>
            ) : null}
            {status === "running" ? (
              <>
                <button className="button button-secondary" type="button" onClick={() => void runAction("stop")} disabled={pendingAction !== null}>
                  <Square size={17} aria-hidden="true" />{pendingAction === "stop" ? "正在停止" : "停止"}
                </button>
                <button className="button button-secondary" type="button" onClick={() => void runAction("restart")} disabled={pendingAction !== null}>
                  <RotateCcw size={17} aria-hidden="true" />{pendingAction === "restart" ? "正在重启" : "重启"}
                </button>
              </>
            ) : null}
            {!hasLifecycleActions ? <p className="muted">当前状态只支持刷新和删除。</p> : null}
          </div>
        </section>

        <section className="detail-section logs-section" aria-labelledby="logs-title">
          <div className="section-heading">
            <div><h2 id="logs-title">最近日志</h2><p>最近 100 行</p></div>
            <button className="icon-button" type="button" onClick={() => void loadLogs()} disabled={logsLoading} aria-label="重新加载日志" title="重新加载日志">
              <RefreshCw className={logsLoading ? "spin" : undefined} size={18} aria-hidden="true" />
            </button>
          </div>
          {logsError ? <ErrorState message={logsError} onRetry={() => void loadLogs()} /> : null}
          {logsLoading ? <LoadingState label="正在加载日志" /> : null}
          {!logsLoading && !logsError && !logs ? <EmptyState title="暂无日志" /> : null}
          {!logsLoading && !logsError && logs ? <pre className="logs" tabIndex={0}>{logs}</pre> : null}
        </section>
      </div>

      <dialog ref={deleteDialog} className="confirm-dialog" aria-labelledby="delete-title">
        <form onSubmit={confirmDelete}>
          <h2 id="delete-title">删除应用？</h2>
          <p>删除后将无法恢复应用 <strong>{app.name}</strong>。</p>
          <div className="form-actions">
            <button className="button button-secondary" type="button" onClick={() => deleteDialog.current?.close()} disabled={pendingAction === "delete"}>取消</button>
            <button className="button button-danger" type="submit" disabled={pendingAction === "delete"}>
              {pendingAction === "delete" ? "正在删除" : "确认删除"}
            </button>
          </div>
        </form>
      </dialog>
    </section>
  );
}
