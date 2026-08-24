import { ArrowRight, Eye, EyeOff, KeyRound } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { errorMessage } from "../../../lib/api";
import { bytCloudErrorMessage, getBytCloudAuthUrl, login } from "../api";
import { setSessionToken } from "../session";
import { AuthLayout } from "./AuthLayout";

interface LoginErrors {
  username?: string;
  password?: string;
}

export function LoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<LoginErrors>({});
  const [formError, setFormError] = useState("");
  const [authError, setAuthError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [startingAuth, setStartingAuth] = useState(false);

  async function startBytCloudAuth() {
    if (startingAuth) return;
    setStartingAuth(true);
    setAuthError("");
    try {
      window.location.assign(await getBytCloudAuthUrl());
    } catch (error) {
      setAuthError(bytCloudErrorMessage(error));
      setStartingAuth(false);
    }
  }

  async function submitLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;

    const nextErrors: LoginErrors = {};
    if (!username.trim()) nextErrors.username = "请输入用户名。";
    if (!password) nextErrors.password = "请输入密码。";
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    setSubmitting(true);
    setFormError("");
    try {
      const session = await login(username.trim(), password);
      setSessionToken(session.token);
      navigate("/apps", { replace: true });
    } catch (error) {
      setFormError(errorMessage(error, "登录失败，请重试。"));
      setSubmitting(false);
    }
  }

  return (
    <AuthLayout title="登录 Astro" subtitle="继续管理你的应用">
      <div className="primary-auth-action">
        <button className="button button-auth" type="button" onClick={startBytCloudAuth} disabled={startingAuth}>
          <KeyRound size={19} aria-hidden="true" />
          {startingAuth ? "正在前往 BytCloud 登录" : "使用 BytCloud 登录"}
          <ArrowRight size={18} aria-hidden="true" />
        </button>
        <p>已有用户可直接登录，首次使用将自动创建 Astro 账号。</p>
        {authError ? (
          <div className="feedback feedback-error" role="alert">
            {authError}
          </div>
        ) : null}
      </div>

      <div className="auth-divider"><span>本地账号密码登录</span></div>

      {searchParams.get("registered") === "1" ? (
        <div className="feedback feedback-success" role="status">
          账号已创建，请登录。
        </div>
      ) : null}

      <form className="form-stack" onSubmit={submitLogin} noValidate>
        <div className="field">
          <label htmlFor="username">用户名</label>
          <input
            id="username"
            name="username"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            aria-invalid={Boolean(errors.username)}
            aria-describedby={errors.username ? "username-error" : undefined}
            required
          />
          {errors.username ? <p className="field-error" id="username-error">{errors.username}</p> : null}
        </div>
        <div className="field">
          <label htmlFor="password">密码</label>
          <div className="password-input">
            <input
              id="password"
              name="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              aria-invalid={Boolean(errors.password)}
              aria-describedby={errors.password ? "password-error" : undefined}
              required
            />
            <button
              className="password-toggle"
              type="button"
              onClick={() => setShowPassword((visible) => !visible)}
              aria-label={showPassword ? "隐藏密码" : "显示密码"}
              title={showPassword ? "隐藏密码" : "显示密码"}
            >
              {showPassword ? <EyeOff size={19} aria-hidden="true" /> : <Eye size={19} aria-hidden="true" />}
            </button>
          </div>
          {errors.password ? <p className="field-error" id="password-error">{errors.password}</p> : null}
        </div>
        {formError ? <div className="feedback feedback-error" role="alert">{formError}</div> : null}
        <button className="button button-secondary button-block" type="submit" disabled={submitting}>
          {submitting ? "正在登录" : "本地账号密码登录"}
        </button>
      </form>

      <p className="auth-footer">没有本地账号？<Link to="/register">注册本地账号</Link></p>
    </AuthLayout>
  );
}
