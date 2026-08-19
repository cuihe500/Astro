import { Eye, EyeOff } from "lucide-react";
import { type FormEvent, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { errorMessage } from "../../../lib/api";
import { login, register } from "../api";
import { setSessionToken } from "../session";
import { AuthLayout } from "./AuthLayout";

interface RegisterErrors {
  username?: string;
  email?: string;
  password?: string;
}

export function RegisterPage() {
  const navigate = useNavigate();
  const emailInput = useRef<HTMLInputElement>(null);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<RegisterErrors>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submitRegistration(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;

    const nextErrors: RegisterErrors = {};
    if (!username.trim()) nextErrors.username = "请输入用户名。";
    if (!email.trim()) nextErrors.email = "请输入邮箱。";
    else if (emailInput.current?.validity.typeMismatch) nextErrors.email = "请输入有效的邮箱地址。";
    if (!password) nextErrors.password = "请输入密码。";
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    setSubmitting(true);
    setFormError("");
    let accountCreated = false;
    try {
      await register(username.trim(), password, email.trim());
      accountCreated = true;
      const session = await login(username.trim(), password);
      setSessionToken(session.token);
      navigate("/apps", { replace: true });
    } catch (error) {
      setFormError(
        accountCreated ? "账号已创建，但自动登录失败。请返回登录页重试。" : errorMessage(error, "注册失败，请重试。"),
      );
      setSubmitting(false);
    }
  }

  return (
    <AuthLayout title="注册备用账号" subtitle="BytCloud Auth 不可用时仍可登录">
      <form className="form-stack" onSubmit={submitRegistration} noValidate>
        <div className="field">
          <label htmlFor="register-username">用户名</label>
          <input
            id="register-username"
            name="username"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            aria-invalid={Boolean(errors.username)}
            aria-describedby={errors.username ? "register-username-error" : undefined}
            required
          />
          {errors.username ? <p className="field-error" id="register-username-error">{errors.username}</p> : null}
        </div>
        <div className="field">
          <label htmlFor="register-email">邮箱</label>
          <input
            ref={emailInput}
            id="register-email"
            name="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            aria-invalid={Boolean(errors.email)}
            aria-describedby={errors.email ? "register-email-error" : undefined}
            required
          />
          {errors.email ? <p className="field-error" id="register-email-error">{errors.email}</p> : null}
        </div>
        <div className="field">
          <label htmlFor="register-password">密码</label>
          <div className="password-input">
            <input
              id="register-password"
              name="password"
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              aria-invalid={Boolean(errors.password)}
              aria-describedby={errors.password ? "register-password-error" : undefined}
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
          {errors.password ? <p className="field-error" id="register-password-error">{errors.password}</p> : null}
        </div>
        {formError ? <div className="feedback feedback-error" role="alert">{formError}</div> : null}
        <button className="button button-primary button-block" type="submit" disabled={submitting}>
          {submitting ? "正在创建账号" : "创建并登录"}
        </button>
      </form>
      <p className="auth-footer">已有账号？<Link to="/login">返回登录</Link></p>
    </AuthLayout>
  );
}
