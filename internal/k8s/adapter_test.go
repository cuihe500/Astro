package k8s

import (
	"context"
	"errors"
	"testing"

	"github.com/cuihe500/astro/internal/model"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestDeleteNamespaceIsIdempotent(t *testing.T) {
	client := useFakeClient(t)
	adapter := NewClientGoAdapter()
	if err := adapter.DeleteNamespace(context.Background(), "missing"); err != nil {
		t.Fatalf("删除不存在的 Namespace 应成功: %v", err)
	}
	if _, err := client.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatalf("创建测试 Namespace 失败: %v", err)
	}
	if err := adapter.DeleteNamespace(context.Background(), "demo"); err != nil {
		t.Fatalf("删除 Namespace 失败: %v", err)
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "demo", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("Namespace 删除后仍存在: %v", err)
	}
}

func TestCreateAppDeletesDeploymentWhenServiceCreationFails(t *testing.T) {
	client := useFakeClient(t)
	client.PrependReactor("create", "services", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("service unavailable")
	})
	err := NewClientGoAdapter().CreateApp(context.Background(), AppSpec{
		Name: "demo", Namespace: "test", Image: "nginx:alpine", Replicas: 1,
		Config: model.AppConfig{Ports: []model.AppPort{{
			Name: "default", ContainerPort: 80, Protocol: "TCP", ServicePort: pointer(int32(80)),
		}}},
	})
	if err == nil {
		t.Fatal("Service 创建失败时应返回错误")
	}
	if _, err := client.AppsV1().Deployments("test").Get(context.Background(), "demo", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("Service 创建失败后 Deployment 未清理: %v", err)
	}
}

func TestBuildDeploymentAndServiceMapsAppConfig(t *testing.T) {
	emptyValue := ""
	servicePort := int32(53)
	falseValue := false
	trueValue := true
	userID := int64(1000)
	groupID := int64(2000)
	terminationGrace := int64(30)
	spec := AppSpec{
		Name: "demo", Namespace: "test", Image: "example/demo:v1", Replicas: 2,
		Config: model.AppConfig{
			Command: []string{"/bin/demo"}, Args: []string{"serve"}, WorkingDir: "/app", ImagePullPolicy: "Never",
			Env: []model.EnvVar{
				{Name: "EMPTY", Value: &emptyValue},
				{Name: "TOKEN", ValueFrom: &model.EnvVarSource{SecretKeyRef: &model.KeyReference{Name: "app-secret", Key: "token"}}},
			},
			EnvFrom: []model.EnvFromSource{{Prefix: "APP_", ConfigMapRef: &model.NamedResourceSource{Name: "app-config"}}},
			Resources: &model.ResourceRequirements{
				Requests: model.ResourceValues{CPU: "100m", Memory: "64Mi"}, Limits: model.ResourceValues{CPU: "1", Memory: "128Mi"},
			},
			Ports: []model.AppPort{
				{Name: "metrics", ContainerPort: 9090, Protocol: "TCP"},
				{Name: "dns", ContainerPort: 5353, Protocol: "UDP", ServicePort: &servicePort},
			},
			StartupProbe: &model.Probe{Exec: &model.ExecAction{Command: []string{"/bin/check"}}, FailureThreshold: 5},
			ReadinessProbe: &model.Probe{HTTPGet: &model.HTTPGetAction{
				Path: "/ready", Port: 8080, Scheme: "HTTPS", HTTPHeaders: []model.HTTPHeader{{Name: "X-Probe", Value: "ready"}},
			}},
			LivenessProbe: &model.Probe{TCPSocket: &model.TCPSocketAction{Port: 8080}},
			Volumes: []model.Volume{
				{Name: "cache", EmptyDir: &model.EmptyDirVolumeSource{Medium: "Memory", SizeLimit: "16Mi"}},
				{Name: "data", PersistentVolumeClaim: &model.PersistentVolumeClaimSource{ClaimName: "app-data", ReadOnly: true}},
				{Name: "config", ConfigMap: &model.NamedResourceSource{Name: "app-config"}},
				{Name: "secret", Secret: &model.NamedResourceSource{Name: "app-secret"}},
			},
			VolumeMounts: []model.VolumeMount{{Name: "data", MountPath: "/data", SubPath: "current", ReadOnly: true}},
			SecurityContext: &model.SecurityContext{
				RunAsNonRoot: &trueValue, RunAsUser: &userID, RunAsGroup: &groupID, FSGroup: &groupID,
				ReadOnlyRootFilesystem: &trueValue, AllowPrivilegeEscalation: &falseValue,
				DropCapabilities: []string{"ALL"}, SeccompProfile: "RuntimeDefault",
			},
			TerminationGracePeriodSeconds: &terminationGrace,
			ImagePullSecrets:              []string{"registry-secret"},
		},
	}

	deployment, err := buildDeployment(spec)
	if err != nil {
		t.Fatalf("构建 Deployment 失败: %v", err)
	}
	pod := deployment.Spec.Template.Spec
	container := pod.Containers[0]
	if container.Command[0] != "/bin/demo" || container.Args[0] != "serve" || container.WorkingDir != "/app" || container.ImagePullPolicy != corev1.PullNever {
		t.Fatal("容器启动配置映射不完整")
	}
	if len(container.Env) != 2 || container.Env[1].ValueFrom.SecretKeyRef.Name != "app-secret" || len(container.EnvFrom) != 1 {
		t.Fatal("环境变量映射不完整")
	}
	if container.Resources.Requests.Cpu().String() != "100m" || container.Resources.Limits.Memory().String() != "128Mi" {
		t.Fatal("资源配置映射错误")
	}
	if len(container.Ports) != 2 || container.ReadinessProbe.HTTPGet.Path != "/ready" || container.LivenessProbe.TCPSocket.Port.IntVal != 8080 {
		t.Fatal("端口或探针映射不完整")
	}
	if len(pod.Volumes) != 4 || len(container.VolumeMounts) != 1 || pod.SecurityContext == nil || *pod.SecurityContext.FSGroup != groupID {
		t.Fatal("存储或 Pod 安全上下文映射不完整")
	}
	if container.SecurityContext == nil || container.SecurityContext.Capabilities.Drop[0] != corev1.Capability("ALL") || len(pod.ImagePullSecrets) != 1 {
		t.Fatal("容器安全上下文或拉取凭据映射不完整")
	}
	service := buildService(spec)
	if service == nil || service.Spec.Type != corev1.ServiceTypeClusterIP || len(service.Spec.Ports) != 1 || service.Spec.Ports[0].Protocol != corev1.ProtocolUDP {
		t.Fatal("多端口 Service 映射错误")
	}
	if service.Spec.Ports[0].Port != servicePort || service.Spec.Ports[0].TargetPort.IntVal != 5353 {
		t.Fatal("Service 端口或目标端口映射错误")
	}
}

func TestValidateAppReferencesUsesMetadataAndDeduplicates(t *testing.T) {
	previous := MetadataClient
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Version: "v1", Kind: "Secret"}, &metav1.PartialObjectMetadata{})
	client := metadatafake.NewSimpleMetadataClient(scheme,
		&metav1.PartialObjectMetadata{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: "test"},
		},
	)
	MetadataClient = client
	t.Cleanup(func() { MetadataClient = previous })
	config := model.AppConfig{
		Env:              []model.EnvVar{{Name: "TOKEN", ValueFrom: &model.EnvVarSource{SecretKeyRef: &model.KeyReference{Name: "app-secret", Key: "token"}}}},
		ImagePullSecrets: []string{"app-secret"},
	}
	if err := NewClientGoAdapter().ValidateAppReferences(context.Background(), "test", config); err != nil {
		t.Fatalf("引用预检失败: %v", err)
	}
	if len(client.Actions()) != 1 {
		t.Fatalf("同一 Secret 应只读取一次元数据，实际 %d 次", len(client.Actions()))
	}
	config.ImagePullSecrets = append(config.ImagePullSecrets, "missing")
	if err := NewClientGoAdapter().ValidateAppReferences(context.Background(), "test", config); err == nil {
		t.Fatal("缺失引用应导致预检失败")
	}
}

func pointer[T any](value T) *T { return &value }

func useFakeClient(t *testing.T) *fake.Clientset {
	t.Helper()
	previousClient := Client
	client := fake.NewSimpleClientset()
	Client = client
	t.Cleanup(func() { Client = previousClient })
	return client
}
