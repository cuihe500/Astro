package k8s

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
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
		Name: "demo", Namespace: "test", Image: "nginx:alpine", Replicas: 1, Port: 80,
	})
	if err == nil {
		t.Fatal("Service 创建失败时应返回错误")
	}
	if _, err := client.AppsV1().Deployments("test").Get(context.Background(), "demo", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("Service 创建失败后 Deployment 未清理: %v", err)
	}
}

func useFakeClient(t *testing.T) *fake.Clientset {
	t.Helper()
	previousClient := Client
	client := fake.NewSimpleClientset()
	Client = client
	t.Cleanup(func() { Client = previousClient })
	return client
}
