const SESSION_TOKEN_KEY = "astro.session.token";

export function getSessionToken(): string | null {
  try {
    return window.localStorage.getItem(SESSION_TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setSessionToken(token: string): void {
  if (!token.trim()) {
    throw new Error("登录响应无效，请重新登录。");
  }

  try {
    window.localStorage.setItem(SESSION_TOKEN_KEY, token);
  } catch {
    throw new Error("浏览器无法保存登录状态，请检查隐私设置后重试。");
  }
}

export function clearSession(): void {
  try {
    window.localStorage.removeItem(SESSION_TOKEN_KEY);
  } catch {
    // localStorage 不可用时，本地也没有可恢复的会话。
  }
}
