package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cuihe500/astro/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// AppSpec 应用规格
type AppSpec struct {
	Name      string
	Namespace string
	Image     string
	Replicas  int32
	Config    model.AppConfig
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
	// DeleteNamespace 删除命名空间
	DeleteNamespace(ctx context.Context, namespace string) error
	// CreateApp 创建应用
	CreateApp(ctx context.Context, spec AppSpec) error
	// ValidateAppReferences 校验应用引用资源存在。
	ValidateAppReferences(ctx context.Context, namespace string, config model.AppConfig) error
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

type appReference struct {
	kind     string
	name     string
	resource schema.GroupVersionResource
}

var (
	configMapResource = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	secretResource    = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	pvcResource       = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
)

// ValidateAppReferences 仅通过元数据 API 校验同命名空间引用。
func (a *ClientGoAdapter) ValidateAppReferences(ctx context.Context, namespace string, config model.AppConfig) error {
	references := collectAppReferences(config)
	if len(references) == 0 {
		return nil
	}
	if MetadataClient == nil {
		return fmt.Errorf("kubernetes metadata client 未初始化")
	}
	for _, reference := range references {
		if _, err := MetadataClient.Resource(reference.resource).Namespace(namespace).
			Get(ctx, reference.name, metav1.GetOptions{}); err != nil {
			return fmt.Errorf("%s %q 不存在或不可访问: %w", reference.kind, reference.name, err)
		}
	}
	return nil
}

func collectAppReferences(config model.AppConfig) []appReference {
	references := make([]appReference, 0)
	seen := make(map[string]bool)
	add := func(kind, name string, resource schema.GroupVersionResource) {
		key := resource.Resource + "/" + name
		if name == "" || seen[key] {
			return
		}
		seen[key] = true
		references = append(references, appReference{kind: kind, name: name, resource: resource})
	}
	for _, env := range config.Env {
		if env.ValueFrom == nil {
			continue
		}
		if env.ValueFrom.ConfigMapKeyRef != nil {
			add("ConfigMap", env.ValueFrom.ConfigMapKeyRef.Name, configMapResource)
		}
		if env.ValueFrom.SecretKeyRef != nil {
			add("Secret", env.ValueFrom.SecretKeyRef.Name, secretResource)
		}
	}
	for _, source := range config.EnvFrom {
		if source.ConfigMapRef != nil {
			add("ConfigMap", source.ConfigMapRef.Name, configMapResource)
		}
		if source.SecretRef != nil {
			add("Secret", source.SecretRef.Name, secretResource)
		}
	}
	for _, volume := range config.Volumes {
		if volume.PersistentVolumeClaim != nil {
			add("PVC", volume.PersistentVolumeClaim.ClaimName, pvcResource)
		}
		if volume.ConfigMap != nil {
			add("ConfigMap", volume.ConfigMap.Name, configMapResource)
		}
		if volume.Secret != nil {
			add("Secret", volume.Secret.Name, secretResource)
		}
	}
	for _, name := range config.ImagePullSecrets {
		add("Secret", name, secretResource)
	}
	return references
}

// EnsureNamespace 确保命名空间存在
func (a *ClientGoAdapter) EnsureNamespace(ctx context.Context, namespace string) error {
	_, err := Client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
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

// DeleteNamespace 删除命名空间，不存在时视为成功。
func (a *ClientGoAdapter) DeleteNamespace(ctx context.Context, namespace string) error {
	err := Client.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// CreateApp 创建应用（Deployment + Service）
func (a *ClientGoAdapter) CreateApp(ctx context.Context, spec AppSpec) error {
	deployment, err := buildDeployment(spec)
	if err != nil {
		return fmt.Errorf("构建 Deployment 失败: %w", err)
	}

	_, err = Client.AppsV1().Deployments(spec.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建 Deployment 失败: %w", err)
	}

	service := buildService(spec)
	if service != nil {
		_, err = Client.CoreV1().Services(spec.Namespace).Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			cleanupErr := Client.AppsV1().Deployments(spec.Namespace).
				Delete(context.WithoutCancel(ctx), spec.Name, metav1.DeleteOptions{})
			if cleanupErr != nil && !apierrors.IsNotFound(cleanupErr) {
				return fmt.Errorf("创建 Service 失败: %w；回滚 Deployment 失败: %v", err, cleanupErr)
			}
			return fmt.Errorf("创建 Service 失败: %w", err)
		}
	}

	return nil
}

func buildDeployment(spec AppSpec) (*appsv1.Deployment, error) {
	labels := appLabels(spec)
	container, err := buildContainer(spec)
	if err != nil {
		return nil, err
	}
	podSpec := corev1.PodSpec{Containers: []corev1.Container{container}}
	podSpec.Volumes, err = buildVolumes(spec.Config.Volumes)
	if err != nil {
		return nil, err
	}
	for _, name := range spec.Config.ImagePullSecrets {
		podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, corev1.LocalObjectReference{Name: name})
	}
	podSpec.TerminationGracePeriodSeconds = spec.Config.TerminationGracePeriodSeconds
	if security := spec.Config.SecurityContext; security != nil && security.FSGroup != nil {
		podSpec.SecurityContext = &corev1.PodSecurityContext{FSGroup: security.FSGroup}
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &spec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": spec.Name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       podSpec,
			},
		},
	}, nil
}

func buildContainer(spec AppSpec) (corev1.Container, error) {
	config := spec.Config
	resources, err := buildResourceRequirements(config.Resources)
	if err != nil {
		return corev1.Container{}, err
	}
	container := corev1.Container{
		Name:            spec.Name,
		Image:           spec.Image,
		Command:         config.Command,
		Args:            config.Args,
		WorkingDir:      config.WorkingDir,
		ImagePullPolicy: corev1.PullPolicy(config.ImagePullPolicy),
		Resources:       resources,
		StartupProbe:    buildProbe(config.StartupProbe),
		ReadinessProbe:  buildProbe(config.ReadinessProbe),
		LivenessProbe:   buildProbe(config.LivenessProbe),
	}
	for _, env := range config.Env {
		variable := corev1.EnvVar{Name: env.Name}
		if env.Value != nil {
			variable.Value = *env.Value
		}
		if env.ValueFrom != nil {
			variable.ValueFrom = &corev1.EnvVarSource{}
			if env.ValueFrom.ConfigMapKeyRef != nil {
				reference := env.ValueFrom.ConfigMapKeyRef
				variable.ValueFrom.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: reference.Name}, Key: reference.Key,
				}
			}
			if env.ValueFrom.SecretKeyRef != nil {
				reference := env.ValueFrom.SecretKeyRef
				variable.ValueFrom.SecretKeyRef = &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: reference.Name}, Key: reference.Key,
				}
			}
		}
		container.Env = append(container.Env, variable)
	}
	for _, source := range config.EnvFrom {
		envFrom := corev1.EnvFromSource{Prefix: source.Prefix}
		if source.ConfigMapRef != nil {
			envFrom.ConfigMapRef = &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: source.ConfigMapRef.Name},
			}
		}
		if source.SecretRef != nil {
			envFrom.SecretRef = &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: source.SecretRef.Name},
			}
		}
		container.EnvFrom = append(container.EnvFrom, envFrom)
	}
	for _, port := range config.Ports {
		container.Ports = append(container.Ports, corev1.ContainerPort{
			Name: port.Name, ContainerPort: port.ContainerPort, Protocol: corev1.Protocol(port.Protocol),
		})
	}
	for _, mount := range config.VolumeMounts {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name: mount.Name, MountPath: mount.MountPath, SubPath: mount.SubPath, ReadOnly: mount.ReadOnly,
		})
	}
	container.SecurityContext = buildContainerSecurityContext(config.SecurityContext)
	return container, nil
}

func buildResourceRequirements(resources *model.ResourceRequirements) (corev1.ResourceRequirements, error) {
	if resources == nil {
		return corev1.ResourceRequirements{}, nil
	}
	requests, err := buildResourceList(resources.Requests)
	if err != nil {
		return corev1.ResourceRequirements{}, err
	}
	limits, err := buildResourceList(resources.Limits)
	if err != nil {
		return corev1.ResourceRequirements{}, err
	}
	return corev1.ResourceRequirements{Requests: requests, Limits: limits}, nil
}

func buildResourceList(values model.ResourceValues) (corev1.ResourceList, error) {
	resources := corev1.ResourceList{}
	for _, item := range []struct {
		name  corev1.ResourceName
		value string
	}{{corev1.ResourceCPU, values.CPU}, {corev1.ResourceMemory, values.Memory}} {
		if item.value == "" {
			continue
		}
		quantity, err := apiresource.ParseQuantity(item.value)
		if err != nil {
			return nil, fmt.Errorf("资源 %s 无效: %w", item.name, err)
		}
		resources[item.name] = quantity
	}
	if len(resources) == 0 {
		return nil, nil
	}
	return resources, nil
}

func buildProbe(probe *model.Probe) *corev1.Probe {
	if probe == nil {
		return nil
	}
	result := &corev1.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}
	if probe.HTTPGet != nil {
		httpGet := probe.HTTPGet
		result.HTTPGet = &corev1.HTTPGetAction{
			Path: httpGet.Path, Port: intstr.FromInt32(httpGet.Port), Scheme: corev1.URIScheme(httpGet.Scheme),
		}
		for _, header := range httpGet.HTTPHeaders {
			result.HTTPGet.HTTPHeaders = append(result.HTTPGet.HTTPHeaders, corev1.HTTPHeader{Name: header.Name, Value: header.Value})
		}
	}
	if probe.TCPSocket != nil {
		result.TCPSocket = &corev1.TCPSocketAction{Port: intstr.FromInt32(probe.TCPSocket.Port)}
	}
	if probe.Exec != nil {
		result.Exec = &corev1.ExecAction{Command: probe.Exec.Command}
	}
	return result
}

func buildVolumes(volumes []model.Volume) ([]corev1.Volume, error) {
	result := make([]corev1.Volume, 0, len(volumes))
	for _, volume := range volumes {
		item := corev1.Volume{Name: volume.Name}
		switch {
		case volume.EmptyDir != nil:
			item.EmptyDir = &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMedium(volume.EmptyDir.Medium)}
			if volume.EmptyDir.SizeLimit != "" {
				quantity, err := apiresource.ParseQuantity(volume.EmptyDir.SizeLimit)
				if err != nil {
					return nil, fmt.Errorf("emptyDir size_limit 无效: %w", err)
				}
				item.EmptyDir.SizeLimit = &quantity
			}
		case volume.PersistentVolumeClaim != nil:
			item.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volume.PersistentVolumeClaim.ClaimName, ReadOnly: volume.PersistentVolumeClaim.ReadOnly,
			}
		case volume.ConfigMap != nil:
			item.ConfigMap = &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: volume.ConfigMap.Name},
			}
		case volume.Secret != nil:
			item.Secret = &corev1.SecretVolumeSource{SecretName: volume.Secret.Name}
		default:
			return nil, fmt.Errorf("卷 %s 缺少来源", volume.Name)
		}
		result = append(result, item)
	}
	return result, nil
}

func buildContainerSecurityContext(security *model.SecurityContext) *corev1.SecurityContext {
	if security == nil || (security.RunAsNonRoot == nil && security.RunAsUser == nil && security.RunAsGroup == nil &&
		security.ReadOnlyRootFilesystem == nil && security.AllowPrivilegeEscalation == nil &&
		len(security.DropCapabilities) == 0 && security.SeccompProfile == "") {
		return nil
	}
	result := &corev1.SecurityContext{
		RunAsNonRoot: security.RunAsNonRoot, RunAsUser: security.RunAsUser, RunAsGroup: security.RunAsGroup,
		ReadOnlyRootFilesystem:   security.ReadOnlyRootFilesystem,
		AllowPrivilegeEscalation: security.AllowPrivilegeEscalation,
	}
	if len(security.DropCapabilities) > 0 {
		result.Capabilities = &corev1.Capabilities{}
		for _, capability := range security.DropCapabilities {
			result.Capabilities.Drop = append(result.Capabilities.Drop, corev1.Capability(capability))
		}
	}
	if security.SeccompProfile != "" {
		result.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileType(security.SeccompProfile)}
	}
	return result
}

func buildService(spec AppSpec) *corev1.Service {
	ports := make([]corev1.ServicePort, 0, len(spec.Config.Ports))
	for _, port := range spec.Config.Ports {
		if port.ServicePort == nil {
			continue
		}
		ports = append(ports, corev1.ServicePort{
			Name: port.Name, Port: *port.ServicePort, Protocol: corev1.Protocol(port.Protocol),
			TargetPort: intstr.FromInt32(port.ContainerPort),
		})
	}
	if len(ports) == 0 {
		return nil
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: appLabels(spec)},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{"app": spec.Name}, Ports: ports,
		},
	}
}

func appLabels(spec AppSpec) map[string]string {
	labels := map[string]string{"app": spec.Name, "managed-by": "astro"}
	for key, value := range spec.Labels {
		labels[key] = value
	}
	return labels
}

// DeleteApp 删除应用
func (a *ClientGoAdapter) DeleteApp(ctx context.Context, name, namespace string) error {
	// 删除 Deployment
	err := Client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("删除 Deployment 失败: %w", err)
	}

	// 删除 Service（忽略不存在的错误）
	err = Client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
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
		if apierrors.IsNotFound(err) {
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

	buf := new(bytes.Buffer)
	_, copyErr := io.Copy(buf, stream)
	closeErr := stream.Close()
	if copyErr != nil {
		if closeErr != nil {
			return "", fmt.Errorf("读取日志失败: %w；关闭日志流失败: %w", copyErr, closeErr)
		}
		return "", fmt.Errorf("读取日志失败: %w", copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("关闭日志流失败: %w", closeErr)
	}

	return buf.String(), nil
}

// Adapter 全局适配器实例
var Adapter AppAdapter = NewClientGoAdapter()
