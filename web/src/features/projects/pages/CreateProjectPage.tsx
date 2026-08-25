import { ArrowLeft } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { errorMessage } from "../../../lib/api";
import { projectAppsPath, projectsPath } from "../../../lib/routes";
import { createProject } from "../api";

export function CreateProjectPage() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [nameError, setNameError] = useState("");
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submitProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;

    const trimmedName = name.trim();
    if (!trimmedName || trimmedName.length > 64) {
      setNameError("请输入 1-64 个字符的项目名称。");
      return;
    }

    setNameError("");
    setFormError("");
    setSubmitting(true);
    try {
      const project = await createProject({ name: trimmedName });
      navigate(projectAppsPath(project.id), { replace: true, state: { message: "项目已创建。" } });
    } catch (requestError) {
      setFormError(errorMessage(requestError, "创建项目失败，请重试。"));
      setSubmitting(false);
    }
  }

  return (
    <section className="page page-narrow" aria-labelledby="create-project-title">
      <Link className="back-link" to={projectsPath}>
        <ArrowLeft size={17} aria-hidden="true" />返回项目列表
      </Link>
      <header className="page-header">
        <div>
          <p className="eyebrow">新项目</p>
          <h1 id="create-project-title">创建项目</h1>
        </div>
      </header>

      <form className="form-stack create-form" onSubmit={submitProject} noValidate>
        <div className="field">
          <label htmlFor="project-name">项目名称</label>
          <input
            id="project-name"
            name="name"
            autoComplete="off"
            maxLength={64}
            value={name}
            onChange={(event) => setName(event.target.value)}
            aria-invalid={Boolean(nameError)}
            aria-describedby={nameError ? "project-name-error" : "project-name-help"}
            required
          />
          <p className="field-help" id="project-name-help">同一账号下不能使用重复名称。</p>
          {nameError ? <p className="field-error" id="project-name-error">{nameError}</p> : null}
        </div>
        {formError ? <div className="feedback feedback-error" role="alert">{formError}</div> : null}
        <div className="form-actions">
          <Link className="button button-secondary" to={projectsPath}>取消</Link>
          <button className="button button-primary" type="submit" disabled={submitting}>
            {submitting ? "正在创建" : "创建项目"}
          </button>
        </div>
      </form>
    </section>
  );
}
