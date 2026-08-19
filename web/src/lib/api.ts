import { clearSession, getSessionToken } from "../features/auth/session";

const API_ROOT = "/api/v1";
const AUTHENTICATION_ERROR_CODES = new Set([10002, 20011, 20012]);
const NETWORK_ERROR_MESSAGE = "无法连接服务器，请检查网络后重试。";
const INVALID_RESPONSE_MESSAGE = "服务器返回了无法识别的响应，请稍后重试。";

export interface ApiResponse<T> {
  code: number;
  message: string;
  data?: T;
}

export class ApiError extends Error {
  readonly code: number;

  constructor(code: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
  }
}

interface RequestOptions extends RequestInit {
  auth?: boolean;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function isAuthenticationError(code: number): boolean {
  return AUTHENTICATION_ERROR_CODES.has(code);
}

export function unwrapApiResponse<T>(payload: unknown): T {
  if (!isRecord(payload) || typeof payload.code !== "number" || typeof payload.message !== "string") {
    throw new ApiError(-1, INVALID_RESPONSE_MESSAGE);
  }

  if (payload.code !== 0) {
    throw new ApiError(payload.code, payload.message || "请求失败，请重试。");
  }

  return payload.data as T;
}

function buildUrl(path: string): string {
  const configuredBase = import.meta.env.VITE_API_BASE_URL?.trim().replace(/\/$/, "") ?? "";
  return `${configuredBase}${API_ROOT}${path}`;
}

function endExpiredSession(): void {
  if (window.location.pathname !== "/login") {
    window.location.assign("/login");
  }
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { auth = false, headers: initialHeaders, ...requestOptions } = options;
  const headers = new Headers(initialHeaders);

  if (requestOptions.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (auth) {
    const token = getSessionToken();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }

  let response: Response;
  try {
    response = await fetch(buildUrl(path), { ...requestOptions, headers });
  } catch {
    throw new ApiError(-1, NETWORK_ERROR_MESSAGE);
  }

  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    throw new ApiError(-1, INVALID_RESPONSE_MESSAGE);
  }

  try {
    return unwrapApiResponse<T>(payload);
  } catch (error) {
    if (error instanceof ApiError && isAuthenticationError(error.code)) {
      clearSession();
      if (auth) {
        endExpiredSession();
      }
    }
    throw error;
  }
}

export function errorMessage(error: unknown, fallback = "请求失败，请重试。"): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
