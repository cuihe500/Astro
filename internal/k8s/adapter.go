package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// AppSpec 应用规格
type AppSpec struct {
	Name      string
	Namespace string
	Image     string
	Replicas  int32
	Port      int32
	Labels    map[string]string
}

// AppStatus 应用状态
type AppStatus struct {
	Status        string // pending/running/stopped/starting/restarting/unknown
	ReadyReplicas int32
	Replicas      int32
	Pods          []PodInfo
}

// PodInfo Pod 信息
type PodInfo struct {
	Name   string
	Status string
	Ready  bool
}

// AppAdapter K8s 应用适配器接口
type AppAdapter interface {
	// EnsureNamespace 确保命名空间存在
	EnsureNamespace(ctx context.Context, namespace string) error
	// CreateApp 创建应用
	CreateApp(ctx context.Context, spec AppSpec) error
	// DeleteApp 删除应用
	DeleteApp(ctx context.Context, name, namespace string) error
	// ScaleApp 调整副本数
	ScaleApp(ctx context.Context, name, namespace string, replicas int32) error
	// GetAppStatus 获取应用状态
	GetAppStatus(ctx context.Context, name, namespace string) (*AppStatus, error)
	// RestartApp 滚动重启应用
	RestartApp(ctx context.Context, name, namespace string) error
	// GetAppLogs 获取应用日志
	GetAppLogs(ctx context.Context, name, namespace string, lines int64) (string, error)
}

// ClientGoAdapter 基于 client-go 的适配器实现
type ClientGoAdapter struct{}

// NewClientGoAdapter 创建 ClientGoAdapter
func NewClientGoAdapter() *ClientGoAdapter {
	return &ClientGoAdapter{}
}

// EnsureNamespace 确保命名空间存在
func (a *ClientGoAdapter) EnsureNamespace(ctx context.Context, namespace string) error {
	_, err := Client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"managed-by": "astro",
			},
		},
	}
	_, err = Client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

// CreateApp 创建应用（Deployment + Service）
func (a *ClientGoAdapter) CreateApp(ctx context.Context, spec AppSpec) error {
	// 确保命名空间存在
	if err := a.EnsureNamespace(ctx, spec.Namespace); err != nil {
		return fmt.Errorf("创建命名空间失败: %w", err)
	}

	// 构建标签
	labels := map[string]string{
		"app":        spec.Name,
		"managed-by": "astro",
	}
	for k, v := range spec.Labels {
		labels[k] = v
	}

	// 创建 Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": spec.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  spec.Name,
							Image: spec.Image,
						},
					},
				},
			},
		},
	}

	// 如果指定了端口，添加端口配置
	if spec.Port > 0 {
		deployment.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				ContainerPort: spec.Port,
			},
		}
	}

	_, err := Client.AppsV1().Deployments(spec.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建 Deployment 失败: %w", err)
	}

	// 如果有端口，创建 Service
	if spec.Port > 0 {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      spec.Name,
				Namespace: spec.Namespace,
				Labels:    labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": spec.Name,
				},
				Ports: []corev1.ServicePort{
					{
						Port:       spec.Port,
						TargetPort: intstr.FromInt32(spec.Port),
					},
				},
			},
		}
		_, err = Client.CoreV1().Services(spec.Namespace).Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("创建 Service 失败: %w", err)
		}
	}

	return nil
}

// DeleteApp 删除应用
func (a *ClientGoAdapter) DeleteApp(ctx context.Context, name, namespace string) error {
	// 删除 Deployment
	err := Client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("删除 Deployment 失败: %w", err)
	}

	// 删除 Service（忽略不存在的错误）
	err = Client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("删除 Service 失败: %w", err)
	}

	return nil
}

// ScaleApp 调整副本数
func (a *ClientGoAdapter) ScaleApp(ctx context.Context, name, namespace string, replicas int32) error {
	deployment, err := Client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	deployment.Spec.Replicas = &replicas
	_, err = Client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("更新副本数失败: %w", err)
	}

	return nil
}

// GetAppStatus 获取应用状态
func (a *ClientGoAdapter) GetAppStatus(ctx context.Context, name, namespace string) (*AppStatus, error) {
	deployment, err := Client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return &AppStatus{Status: "unknown"}, nil
		}
		return nil, fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	// 获取 Pod 列表
	pods, err := Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", name),
	})
	if err != nil {
		return nil, fmt.Errorf("获取 Pod 列表失败: %w", err)
	}

	podInfos := make([]PodInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
		ready := false
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
		podInfos = append(podInfos, PodInfo{
			Name:   pod.Name,
			Status: string(pod.Status.Phase),
			Ready:  ready,
		})
	}

	// 确定应用状态
	status := a.determineStatus(deployment)

	return &AppStatus{
		Status:        status,
		ReadyReplicas: deployment.Status.ReadyReplicas,
		Replicas:      *deployment.Spec.Replicas,
		Pods:          podInfos,
	}, nil
}

// determineStatus 根据 Deployment 状态确定应用状态
func (a *ClientGoAdapter) determineStatus(deployment *appsv1.Deployment) string {
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas == 0 {
		return "stopped"
	}

	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return "running"
	}

	if deployment.Status.ReadyReplicas == 0 {
		return "pending"
	}

	return "starting"
}

// RestartApp 滚动重启应用
func (a *ClientGoAdapter) RestartApp(ctx context.Context, name, namespace string) error {
	deployment, err := Client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	// 通过修改 annotation 触发滚动重启
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = Client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("重启 Deployment 失败: %w", err)
	}

	return nil
}

// GetAppLogs 获取应用日志
func (a *ClientGoAdapter) GetAppLogs(ctx context.Context, name, namespace string, lines int64) (string, error) {
	// 获取应用的 Pod 列表
	pods, err := Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", name),
	})
	if err != nil {
		return "", fmt.Errorf("获取 Pod 列表失败: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("没有找到运行中的 Pod")
	}

	// 获取第一个 Pod 的日志
	podName := pods.Items[0].Name
	req := Client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		TailLines: &lines,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("获取日志流失败: %w", err)
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return "", fmt.Errorf("读取日志失败: %w", err)
	}

	return buf.String(), nil
}

// Adapter 全局适配器实例
var Adapter AppAdapter = NewClientGoAdapter()
