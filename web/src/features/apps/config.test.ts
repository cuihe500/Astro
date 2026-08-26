import { describe, expect, it } from "vitest";
import { buildAppConfig, emptyAppConfigDraft, parseAppConfig } from "./config";

describe("应用高级配置", () => {
  it("把表单草稿转换为受控请求", () => {
    const draft = emptyAppConfigDraft();
    draft.command = "/bin/sh";
    draft.args = "-c\necho ready";
    draft.env = [{ id: 1, name: "TOKEN", source: "secret", value: "", refName: "app-secret", refKey: "token" }];
    draft.ports = [{ id: 2, name: "http", containerPort: "8080", protocol: "TCP", servicePort: "80" }];
    draft.runAsNonRoot = true;
    draft.allowPrivilegeEscalation = true;
    const result = buildAppConfig(draft);
    expect(result.errors).toEqual({});
    expect(result.config).toMatchObject({
      command: ["/bin/sh"], args: ["-c", "echo ready"],
      env: [{ name: "TOKEN", value_from: { secret_key_ref: { name: "app-secret", key: "token" } } }],
      ports: [{ name: "http", container_port: 8080, protocol: "TCP", service_port: 80 }],
      security_context: { run_as_non_root: true, allow_privilege_escalation: false },
    });
  });

  it("拒绝非法动态配置", () => {
    const draft = emptyAppConfigDraft();
    draft.ports = [{ id: 1, name: "HTTP_PORT_TOO_LONG", containerPort: "70000", protocol: "TCP", servicePort: "" }];
    expect(buildAppConfig(draft).errors["ports.0"]).toBeTruthy();
  });

  it("拒绝服务端返回的错误配置形状", () => {
    expect(parseAppConfig(null)).toEqual({});
    expect(() => parseAppConfig({ image_pull_secrets: [123] })).toThrow();
  });
});
