import { CircleAlert } from "lucide-react";
import { Link, useLoaderData } from "react-router-dom";
import { AuthLayout } from "./AuthLayout";

export interface OAuthCallbackData {
  error: string;
}

export function OAuthCallbackPage() {
  const { error } = useLoaderData() as OAuthCallbackData;

  return (
    <AuthLayout title="BytCloud Auth 未完成" subtitle="你仍可使用账号密码登录">
      <div className="callback-error" role="alert">
        <CircleAlert aria-hidden="true" />
        <p>{error}</p>
      </div>
      <Link className="button button-primary button-block" to="/login">
        返回登录
      </Link>
    </AuthLayout>
  );
}
