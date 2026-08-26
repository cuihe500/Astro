import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { AppConfigDetails } from "../components/AppConfigDetails";
import { AppConfigFields } from "../components/AppConfigFields";
import { emptyAppConfigDraft } from "../config";
import { CreateAppPage } from "./CreateAppPage";

describe("应用高级配置页面", () => {
  it("创建页默认折叠高级配置并提供语义分组", () => {
    const markup = renderToStaticMarkup(
      <MemoryRouter initialEntries={["/projects/3/apps/new"]}>
        <Routes><Route path="/projects/:projectId/apps/new" element={<CreateAppPage />} /></Routes>
      </MemoryRouter>,
    );
    expect(markup).toContain("<details class=\"advanced-config\"");
    expect(markup).not.toContain("<details class=\"advanced-config\" open");
    expect(markup).toContain("高级配置");
    expect(markup).toContain("健康检查");
    expect(markup).toContain("安全与生命周期");
  });

  it("动态行错误与首个无效字段关联", () => {
    const draft = emptyAppConfigDraft();
    draft.env = [{ id: 1, name: "", source: "value", value: "", refName: "", refKey: "" }];
    const markup = renderToStaticMarkup(<AppConfigFields
      draft={draft}
      errors={{ "env.0": "环境变量名称无效。" }}
      detailsRef={{ current: null }}
      onChange={() => undefined}
    />);
    expect(markup).toContain('id="env-1-name"');
    expect(markup).toContain('aria-describedby="env-1-error"');
    expect(markup).toContain('id="env-1-error"');
  });

  it("详情完整显示引用和 HTTP 探针 Header", () => {
    const markup = renderToStaticMarkup(<AppConfigDetails config={{
      env: [{ name: "TOKEN", value_from: { secret_key_ref: { name: "app-secret", key: "token" } } }],
      image_pull_secrets: ["registry-secret"],
      readiness_probe: { http_get: { path: "/ready", port: 8080, http_headers: [{ name: "X-Probe", value: "ready" }] } },
    }} />);
    expect(markup).toContain("Secret app-secret/token");
    expect(markup).toContain("registry-secret");
    expect(markup).toContain("X-Probe: ready");
  });
});
