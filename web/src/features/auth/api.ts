import { ApiError, request } from "../../lib/api";

export const BYTCLOUD_PROVIDER_ALIAS = "bytcloudauth";

interface LoginResponse {
  token: string;
  uuid: string;
}

interface AuthUrlResponse {
  auth_url: string;
}

function requireLoginResponse(data: LoginResponse | undefined): LoginResponse {
  if (!data || typeof data.token !== "string" || !data.token || typeof data.uuid !== "string") {
    throw new ApiError(-1, "登录响应无效，请重新登录。");
  }
  return data;
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  const data = await request<LoginResponse>("/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  return requireLoginResponse(data);
}

export function register(username: string, password: string, email: string): Promise<void> {
  return request<void>("/register", {
    method: "POST",
    body: JSON.stringify({ username, password, email }),
  });
}

export async function getBytCloudAuthUrl(): Promise<string> {
  const data = await request<AuthUrlResponse>(`/oauth2/${BYTCLOUD_PROVIDER_ALIAS}/login`);
  if (!data || typeof data.auth_url !== "string") {
    throw new ApiError(-1, "BytCloud Auth 暂时不可用，请使用账号密码登录。");
  }

  try {
    const url = new URL(data.auth_url);
    if (url.protocol !== "https:" && url.protocol !== "http:") {
      throw new Error("unsupported protocol");
    }
    return url.toString();
  } catch {
    throw new ApiError(-1, "BytCloud Auth 返回了无效地址，请使用账号密码登录。");
  }
}

export async function completeBytCloudAuth(provider: string, code: string, state: string): Promise<LoginResponse> {
  const query = new URLSearchParams({ code, state });
  const data = await request<LoginResponse>(`/oauth2/${encodeURIComponent(provider)}/callback?${query}`);
  return requireLoginResponse(data);
}

export function bytCloudErrorMessage(error: unknown): string {
  return error instanceof Error && error.message ? error.message : "BytCloud Auth 登录失败，请重试。";
}
