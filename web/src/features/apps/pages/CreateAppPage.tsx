import { ArrowLeft } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { errorMessage } from "../../../lib/api";
import { createApp } from "../api";

interface CreateErrors {
  name?: string;
  image?: string;
  replicas?: string;
  port?: string;
}

const APP_NAME_PATTERN = /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/;

export function CreateAppPage() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [image, setImage] = useState("");
  const [replicas, setReplicas] = useState("1");
  const [port, setPort] = useState("");
  const [errors, setErrors] = useState<CreateErrors>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submitApp(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;

    const trimmedName = name.trim();
    const trimmedImage = image.trim();
    const replicasNumber = Number(replicas);
    const portNumber = port === "" ? 0 : Number(port);
    const nextErrors: CreateErrors = {};

    if (!trimmedName) nextErrors.name = "请输入应用名称。";
    else if (trimmedName.length > 63 || !APP_NAME_PATTERN.test(trimmedName)) {
      nextErrors.name = "使用 1-63 位小写字母、数字或短横线，且首尾不能是短横线。";
    }
    if (!trimmedImage) nextErrors.image = "请输入容器镜像。";
    if (!Number.isInteger(replicasNumber) || replicasNumber < 1 || replicasNumber > 10) {
      nextErrors.replicas = "副本数必须是 1-10 的整数。";
    }
    if (port !== "" && (!Number.isInteger(portNumber) || portNumber < 1 || portNumber > 65535)) {
      nextErrors.port = "端口必须是 1-65535 的整数。";
    }

    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    setSubmitting(true);
    setFormError("");
    try {
      const app = await createApp({ name: trimmedName, image: trimmedImage, replicas: replicasNumber, port: portNumber });
      navigate(`/apps/${app.id}`, { replace: true, state: { message: "应用已创建。" } });
    } catch (requestError) {
      setFormError(errorMessage(requestError, "创建应用失败，请重试。"));
      setSubmitting(false);
    }
  }

  return (
    <section className="page page-narrow" aria-labelledby="create-title">
      <Link className="back-link" to="/apps"><ArrowLeft size={17} aria-hidden="true" />返回应用列表</Link>
      <header className="page-header">
        <div>
          <p className="eyebrow">新应用</p>
          <h1 id="create-title">创建应用</h1>
        </div>
      </header>

      <form className="form-stack create-form" onSubmit={submitApp} noValidate>
        <div className="field">
          <label htmlFor="app-name">应用名称</label>
          <input
            id="app-name"
            name="name"
            autoComplete="off"
            maxLength={63}
            placeholder="my-app"
            value={name}
            onChange={(event) => setName(event.target.value)}
            aria-invalid={Boolean(errors.name)}
            aria-describedby={errors.name ? "app-name-error" : "app-name-help"}
            required
          />
          <p className="field-help" id="app-name-help">小写字母、数字和短横线</p>
          {errors.name ? <p className="field-error" id="app-name-error">{errors.name}</p> : null}
        </div>
        <div className="field">
          <label htmlFor="app-image">容器镜像</label>
          <input
            id="app-image"
            name="image"
            autoComplete="off"
            placeholder="nginx:latest"
            value={image}
            onChange={(event) => setImage(event.target.value)}
            aria-invalid={Boolean(errors.image)}
            aria-describedby={errors.image ? "app-image-error" : undefined}
            required
          />
          {errors.image ? <p className="field-error" id="app-image-error">{errors.image}</p> : null}
        </div>
        <div className="form-grid">
          <div className="field">
            <label htmlFor="app-replicas">副本数</label>
            <input
              id="app-replicas"
              name="replicas"
              type="number"
              inputMode="numeric"
              min="1"
              max="10"
              step="1"
              value={replicas}
              onChange={(event) => setReplicas(event.target.value)}
              aria-invalid={Boolean(errors.replicas)}
              aria-describedby={errors.replicas ? "app-replicas-error" : undefined}
              required
            />
            {errors.replicas ? <p className="field-error" id="app-replicas-error">{errors.replicas}</p> : null}
          </div>
          <div className="field">
            <label htmlFor="app-port">端口 <span className="optional">选填</span></label>
            <input
              id="app-port"
              name="port"
              type="number"
              inputMode="numeric"
              min="1"
              max="65535"
              step="1"
              placeholder="80"
              value={port}
              onChange={(event) => setPort(event.target.value)}
              aria-invalid={Boolean(errors.port)}
              aria-describedby={errors.port ? "app-port-error" : undefined}
            />
            {errors.port ? <p className="field-error" id="app-port-error">{errors.port}</p> : null}
          </div>
        </div>
        {formError ? <div className="feedback feedback-error" role="alert">{formError}</div> : null}
        <div className="form-actions">
          <Link className="button button-secondary" to="/apps">取消</Link>
          <button className="button button-primary" type="submit" disabled={submitting}>
            {submitting ? "正在创建" : "创建应用"}
          </button>
        </div>
      </form>
    </section>
  );
}
