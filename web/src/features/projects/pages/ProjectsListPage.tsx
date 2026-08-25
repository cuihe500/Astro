import { ArrowRight, FolderKanban, Plus, RefreshCw, Trash2 } from "lucide-react";
import { type FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { EmptyState, ErrorState } from "../../../components/Feedback";
import { errorMessage } from "../../../lib/api";
import { projectAppsPath } from "../../../lib/routes";
import { deleteProject, getProjects } from "../api";
import type { Project } from "../types";

export function ProjectsListPage() {
  const location = useLocation();
  const deleteDialog = useRef<HTMLDialogElement>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");
  const [deleteError, setDeleteError] = useState("");
  const [feedback, setFeedback] = useState((location.state as { message?: string } | null)?.message ?? "");

  const loadProjects = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setProjects(await getProjects());
    } catch (requestError) {
      setError(errorMessage(requestError, "项目列表加载失败。"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // 首次进入页面时同步远端数据，状态更新来自请求结果。
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadProjects();
  }, [loadProjects]);

  function openDeleteDialog(project: Project) {
    setSelectedProject(project);
    setDeleteError("");
    deleteDialog.current?.showModal();
  }

  async function confirmDelete(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProject || deleting) return;
    setDeleting(true);
    try {
      await deleteProject(selectedProject.id);
      setProjects((current) => current.filter((project) => project.id !== selectedProject.id));
      setFeedback(`项目“${selectedProject.name}”已删除。`);
      deleteDialog.current?.close();
      setSelectedProject(null);
    } catch (requestError) {
      setDeleteError(errorMessage(requestError, "删除项目失败，请重试。"));
    } finally {
      setDeleting(false);
    }
  }

  return (
    <section className="page" aria-labelledby="projects-title">
      <header className="page-header">
        <div>
          <p className="eyebrow">工作台</p>
          <h1 id="projects-title">我的项目</h1>
        </div>
        <div className="page-actions">
          <button className="icon-button" type="button" onClick={() => void loadProjects()} disabled={loading} aria-label="刷新项目列表" title="刷新项目列表">
            <RefreshCw className={loading ? "spin" : undefined} size={19} aria-hidden="true" />
          </button>
          <Link className="button button-primary" to="/projects/new">
            <Plus size={18} aria-hidden="true" />创建项目
          </Link>
        </div>
      </header>

      {feedback ? <div className="feedback feedback-success" role="status">{feedback}</div> : null}
      {error && projects.length === 0 ? <ErrorState message={error} onRetry={() => void loadProjects()} /> : null}
      {loading && projects.length === 0 ? <div className="list-loading" role="status">正在加载项目...</div> : null}
      {!loading && !error && projects.length === 0 ? (
        <EmptyState title="还没有项目">
          <p>先创建一个项目，再在其中部署应用。</p>
          <Link className="button button-primary" to="/projects/new">
            <Plus size={18} aria-hidden="true" />创建第一个项目
          </Link>
        </EmptyState>
      ) : null}

      {projects.length > 0 ? (
        <>
          {error ? <div className="feedback feedback-error" role="alert">{error}</div> : null}
          <div className="project-grid">
            {projects.map((project) => (
              <article className="project-card" key={project.id}>
                <FolderKanban size={24} aria-hidden="true" />
                <div className="project-card-content">
                  <h2>{project.name}</h2>
                </div>
                <div className="project-card-actions">
                  <Link className="button button-secondary" to={projectAppsPath(project.id)}>
                    进入项目<ArrowRight size={17} aria-hidden="true" />
                  </Link>
                  <button className="icon-button icon-button-danger" type="button" onClick={() => openDeleteDialog(project)} aria-label={`删除项目 ${project.name}`} title={`删除项目 ${project.name}`}>
                    <Trash2 size={18} aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))}
          </div>
        </>
      ) : null}

      <dialog ref={deleteDialog} className="confirm-dialog" aria-labelledby="delete-project-title">
        <form onSubmit={confirmDelete}>
          <h2 id="delete-project-title">删除项目？</h2>
          <p>仅空项目可以删除。确认删除 <strong>{selectedProject?.name}</strong>？</p>
          {deleteError ? <div className="feedback feedback-error" role="alert">{deleteError}</div> : null}
          <div className="form-actions">
            <button className="button button-secondary" type="button" onClick={() => deleteDialog.current?.close()} disabled={deleting}>取消</button>
            <button className="button button-danger" type="submit" disabled={deleting}>
              {deleting ? "正在删除" : "确认删除"}
            </button>
          </div>
        </form>
      </dialog>
    </section>
  );
}
