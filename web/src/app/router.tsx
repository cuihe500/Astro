import { replace, redirect, type LoaderFunctionArgs, createBrowserRouter } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { getSessionToken, setSessionToken } from "../features/auth/session";
import { completeBytCloudAuth, bytCloudErrorMessage, BYTCLOUD_PROVIDER_ALIAS } from "../features/auth/api";
import { AppsListPage } from "../features/apps/pages/AppsListPage";
import { AppDetailPage } from "../features/apps/pages/AppDetailPage";
import { CreateAppPage } from "../features/apps/pages/CreateAppPage";
import { LoginPage } from "../features/auth/pages/LoginPage";
import { OAuthCallbackPage, type OAuthCallbackData } from "../features/auth/pages/OAuthCallbackPage";
import { RegisterPage } from "../features/auth/pages/RegisterPage";

export function rootLoader() {
  return redirect(getSessionToken() ? "/apps" : "/login");
}

function publicOnlyLoader() {
  return getSessionToken() ? redirect("/apps") : null;
}

function protectedLoader() {
  return getSessionToken() ? null : redirect("/login");
}

export async function oauthCallbackLoader({ params, request }: LoaderFunctionArgs): Promise<OAuthCallbackData | Response> {
  const provider = params.provider ?? "";
  const url = new URL(request.url);
  if (provider !== BYTCLOUD_PROVIDER_ALIAS) {
    return replace(`/oauth2/${BYTCLOUD_PROVIDER_ALIAS}/callback?error=invalid_provider`);
  }

  const providerError = url.searchParams.get("error");
  if (providerError) {
    return {
      error: "BytCloud Auth 未完成登录，请返回后使用本地账号密码登录。",
    };
  }

  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  if (!code || !state) {
    return { error: "BytCloud Auth 回调信息不完整，请重新尝试。" };
  }

  try {
    const session = await completeBytCloudAuth(provider, code, state);
    setSessionToken(session.token);
    return redirect("/apps");
  } catch (error) {
    return { error: bytCloudErrorMessage(error) };
  }
}

function fallbackLoader() {
  return redirect(getSessionToken() ? "/apps" : "/login");
}

export const router = createBrowserRouter([
  { path: "/", loader: rootLoader },
  { path: "/login", loader: publicOnlyLoader, element: <LoginPage /> },
  { path: "/register", loader: publicOnlyLoader, element: <RegisterPage /> },
  { path: "/oauth2/:provider/callback", loader: oauthCallbackLoader, element: <OAuthCallbackPage /> },
  {
    element: <AppShell />,
    loader: protectedLoader,
    children: [
      { path: "/apps", element: <AppsListPage /> },
      { path: "/apps/new", element: <CreateAppPage /> },
      { path: "/apps/:id", element: <AppDetailPage /> },
    ],
  },
  { path: "*", loader: fallbackLoader },
]);
