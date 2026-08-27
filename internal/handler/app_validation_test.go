package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cuihe500/astro/internal/model"
)

func TestDecodeAndValidateCreateAppRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantError  bool
		assertPort bool
		noPort     bool
	}{
		{name: "最小请求", body: `{"name":"demo","image":"nginx:latest","replicas":1}`},
		{name: "旧端口归一化", body: `{"name":"demo","image":"nginx:latest","replicas":1,"port":8080}`, assertPort: true},
		{name: "旧空端口兼容", body: `{"name":"demo","image":"nginx","replicas":1,"port":0}`, noPort: true},
		{name: "旧空端口仍校验高级配置", body: `{"name":"demo","image":"nginx","replicas":1,"port":0,"config":{"ports":[{"name":"http","container_port":0}]}}`, wantError: true},
		{name: "负端口", body: `{"name":"demo","image":"nginx","replicas":1,"port":-1}`, wantError: true},
		{name: "完整配置", body: `{
			"name":"demo","image":"nginx:latest","replicas":1,
			"config":{
				"command":["/bin/sh"],"args":["-c","echo ready"],"working_dir":"/app","image_pull_policy":"IfNotPresent",
				"env":[{"name":"MODE","value":"production"},{"name":"PASSWORD","value_from":{"secret_key_ref":{"name":"demo-secret","key":"password"}}}],
				"env_from":[{"prefix":"APP_","config_map_ref":{"name":"demo-config"}}],
				"resources":{"requests":{"cpu":"100m","memory":"128Mi"},"limits":{"cpu":"500m","memory":"512Mi"}},
				"ports":[{"name":"http","container_port":8080,"service_port":80}],
				"startup_probe":{"exec":{"command":["/bin/check"]}},
				"readiness_probe":{"http_get":{"path":"/ready","port":8080,"http_headers":[{"name":"X-Probe","value":"ready"}]}},
				"liveness_probe":{"tcp_socket":{"port":8080}},
				"volumes":[{"name":"cache","empty_dir":{"medium":"Memory","size_limit":"64Mi"}},{"name":"data","persistent_volume_claim":{"claim_name":"demo-data"}}],
				"volume_mounts":[{"name":"data","mount_path":"/data"}],
				"security_context":{"run_as_non_root":true,"run_as_user":1000,"allow_privilege_escalation":false,"drop_capabilities":["all"],"seccomp_profile":"RuntimeDefault"},
				"termination_grace_period_seconds":30,"image_pull_secrets":["registry-secret"]
			}}`},
		{name: "未知字段", body: `{"name":"demo","image":"nginx","replicas":1,"pod_spec":{}}`, wantError: true},
		{name: "尾随 JSON", body: `{"name":"demo","image":"nginx","replicas":1}{}`, wantError: true},
		{name: "超过大小", body: `{"name":"demo","image":"` + strings.Repeat("x", maxCreateAppBodyBytes) + `","replicas":1}`, wantError: true},
		{name: "端口冲突", body: `{"name":"demo","image":"nginx","replicas":1,"port":80,"config":{"ports":[{"name":"web","container_port":8080}]}}`, wantError: true},
		{name: "环境变量来源冲突", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"env":[{"name":"VALUE","value":"x","value_from":{"secret_key_ref":{"name":"secret","key":"key"}}}]}}`, wantError: true},
		{name: "危险权限", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"security_context":{"allow_privilege_escalation":true}}}`, wantError: true},
		{name: "未知 capability", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"security_context":{"drop_capabilities":["SUPER_POWER"]}}}`, wantError: true},
		{name: "非法 envFrom 前缀", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"env_from":[{"prefix":"APP-NAME_","config_map_ref":{"name":"demo-config"}}]}}`, wantError: true},
		{name: "资源反向限制", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"resources":{"requests":{"cpu":"2"},"limits":{"cpu":"1"}}}}`, wantError: true},
		{name: "资源数量非正", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"resources":{"requests":{"memory":"0"}}}}`, wantError: true},
		{name: "端口名称重复", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"ports":[{"name":"http","container_port":8080},{"name":"http","container_port":8081}]}}`, wantError: true},
		{name: "Service端口重复", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"ports":[{"name":"http","container_port":8080,"service_port":80},{"name":"admin","container_port":8081,"service_port":80}]}}`, wantError: true},
		{name: "未声明卷", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"volume_mounts":[{"name":"data","mount_path":"/data"}]}}`, wantError: true},
		{name: "卷来源冲突", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"volumes":[{"name":"data","empty_dir":{},"secret":{"name":"app-secret"}}]}}`, wantError: true},
		{name: "挂载子路径越界", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"volumes":[{"name":"data","empty_dir":{}}],"volume_mounts":[{"name":"data","mount_path":"/data","sub_path":"../secret"}]}}`, wantError: true},
		{name: "探针多 handler", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"liveness_probe":{"http_get":{"path":"/","port":80},"tcp_socket":{"port":80}}}}`, wantError: true},
		{name: "Exec探针命令为空", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"liveness_probe":{"exec":{"command":[]}}}}`, wantError: true},
		{name: "HTTP Header换行", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"readiness_probe":{"http_get":{"path":"/ready","port":80,"http_headers":[{"name":"X-Probe","value":"ready\r\nInjected: value"}]}}}}`, wantError: true},
		{name: "imagePullSecret重复", body: `{"name":"demo","image":"nginx","replicas":1,"config":{"image_pull_secrets":["registry-secret","registry-secret"]}}`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := decodeAndValidateCreateAppRequest(strings.NewReader(test.body))
			if test.wantError {
				if err == nil {
					t.Fatal("期望校验失败")
				}
				return
			}
			if err != nil {
				t.Fatalf("请求校验失败: %v", err)
			}
			if test.assertPort {
				port := request.Config.Ports[0]
				if port.Protocol != "TCP" || port.ContainerPort != 8080 || port.ServicePort == nil || *port.ServicePort != 8080 {
					t.Fatalf("旧端口归一化错误: %+v", port)
				}
			}
			if test.noPort && (request.Port != nil || len(request.Config.Ports) != 0) {
				t.Fatalf("旧空端口应视为未配置: %+v", request)
			}
		})
	}
}

func TestAppResponseSerialization(t *testing.T) {
	value := "production"
	app := model.App{
		Name:   "demo",
		Config: model.AppConfig{Env: []model.EnvVar{{Name: "MODE", Value: &value}}},
	}

	listData, err := json.Marshal([]model.App{app})
	if err != nil {
		t.Fatalf("序列化应用列表失败: %v", err)
	}
	if strings.Contains(string(listData), `"config"`) {
		t.Fatalf("应用列表不应包含 config: %s", listData)
	}

	detailData, err := json.Marshal(appDetailResponse(&app))
	if err != nil {
		t.Fatalf("序列化应用详情失败: %v", err)
	}
	if !strings.Contains(string(detailData), `"config":{"env"`) {
		t.Fatalf("应用详情应包含 config: %s", detailData)
	}
}

func TestCreateAppCollectionLimits(t *testing.T) {
	environment := strings.Repeat(`{"name":"A","value":"x"},`, 100) + `{"name":"B","value":"x"}`
	body := fmt.Sprintf(`{"name":"demo","image":"nginx","replicas":1,"config":{"env":[%s]}}`, environment)
	if _, err := decodeAndValidateCreateAppRequest(strings.NewReader(body)); err == nil {
		t.Fatal("环境变量超过上限时应失败")
	}
}
