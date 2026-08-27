package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/cuihe500/astro/internal/model"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
)

const (
	maxCreateAppBodyBytes = 64 << 10
	maxPathLength         = 4096
)

var (
	appNamePattern      = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
	headerNamePattern   = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")
	allowedCapabilities = map[string]bool{
		"ALL": true, "AUDIT_CONTROL": true, "AUDIT_READ": true, "AUDIT_WRITE": true,
		"BLOCK_SUSPEND": true, "BPF": true, "CHECKPOINT_RESTORE": true, "CHOWN": true,
		"DAC_OVERRIDE": true, "DAC_READ_SEARCH": true, "FOWNER": true, "FSETID": true,
		"IPC_LOCK": true, "IPC_OWNER": true, "KILL": true, "LEASE": true,
		"LINUX_IMMUTABLE": true, "MAC_ADMIN": true, "MAC_OVERRIDE": true, "MKNOD": true,
		"NET_ADMIN": true, "NET_BIND_SERVICE": true, "NET_BROADCAST": true, "NET_RAW": true,
		"PERFMON": true, "SETFCAP": true, "SETGID": true, "SETPCAP": true, "SETUID": true,
		"SYS_ADMIN": true, "SYS_BOOT": true, "SYS_CHROOT": true, "SYS_MODULE": true,
		"SYS_NICE": true, "SYS_PACCT": true, "SYS_PTRACE": true, "SYS_RAWIO": true,
		"SYS_RESOURCE": true, "SYS_TIME": true, "SYS_TTY_CONFIG": true, "SYSLOG": true,
		"WAKE_ALARM": true,
	}
)

func decodeCreateAppRequest(body io.Reader) (CreateAppRequest, error) {
	var request CreateAppRequest
	data, err := io.ReadAll(io.LimitReader(body, maxCreateAppBodyBytes+1))
	if err != nil {
		return request, err
	}
	if len(data) > maxCreateAppBodyBytes {
		return request, fmt.Errorf("请求体不能超过 64 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return request, fmt.Errorf("请求体只能包含一个 JSON 对象")
		}
		return request, err
	}
	return request, nil
}

func decodeAndValidateCreateAppRequest(body io.Reader) (CreateAppRequest, error) {
	request, err := decodeCreateAppRequest(body)
	if err != nil {
		return request, err
	}
	if err := normalizeAndValidateCreateAppRequest(&request); err != nil {
		return request, err
	}
	return request, nil
}

func normalizeAndValidateCreateAppRequest(request *CreateAppRequest) error {
	request.Name = strings.TrimSpace(request.Name)
	request.Image = strings.TrimSpace(request.Image)
	if len(request.Name) > 63 || !appNamePattern.MatchString(request.Name) || request.Image == "" || len(request.Image) > 256 {
		return fmt.Errorf("应用名称或容器镜像无效")
	}
	if request.Replicas < 1 || request.Replicas > 10 {
		return fmt.Errorf("副本数必须是 1-10 的整数")
	}
	if request.Port != nil {
		if *request.Port == 0 {
			request.Port = nil
			return validateAppConfig(&request.Config)
		}
		if *request.Port < 1 || *request.Port > 65535 {
			return fmt.Errorf("端口必须是 1-65535 的整数")
		}
		if len(request.Config.Ports) > 0 {
			return fmt.Errorf("port 不能与 config.ports 同时使用")
		}
		port := *request.Port
		request.Config.Ports = []model.AppPort{{
			Name: "default", ContainerPort: port, Protocol: "TCP", ServicePort: &port,
		}}
	}
	return validateAppConfig(&request.Config)
}

func validateAppConfig(config *model.AppConfig) error {
	if len(config.Command) > 20 || len(config.Args) > 100 {
		return fmt.Errorf("command 最多 20 项，args 最多 100 项")
	}
	if len(config.Env)+len(config.EnvFrom) > 100 {
		return fmt.Errorf("环境变量与引用合计最多 100 项")
	}
	if len(config.Ports) > 20 || len(config.Volumes) > 20 || len(config.VolumeMounts) > 20 {
		return fmt.Errorf("端口、卷和挂载分别最多 20 项")
	}
	if len(config.ImagePullSecrets) > 10 {
		return fmt.Errorf("imagePullSecret 最多 10 项")
	}
	if err := validateAbsolutePath(config.WorkingDir, "工作目录", true); err != nil {
		return err
	}
	if config.ImagePullPolicy != "" && config.ImagePullPolicy != "Always" &&
		config.ImagePullPolicy != "IfNotPresent" && config.ImagePullPolicy != "Never" {
		return fmt.Errorf("镜像拉取策略无效")
	}
	for _, item := range config.Command {
		if strings.IndexByte(item, 0) >= 0 {
			return fmt.Errorf("command 不能包含 NUL 字符")
		}
	}
	for _, item := range config.Args {
		if strings.IndexByte(item, 0) >= 0 {
			return fmt.Errorf("args 不能包含 NUL 字符")
		}
	}
	if err := validateEnvironment(config); err != nil {
		return err
	}
	if err := validateResources(config.Resources); err != nil {
		return err
	}
	if err := validatePorts(config.Ports); err != nil {
		return err
	}
	for _, item := range []struct {
		name  string
		probe *model.Probe
		start bool
	}{
		{name: "startup_probe", probe: config.StartupProbe, start: true},
		{name: "readiness_probe", probe: config.ReadinessProbe},
		{name: "liveness_probe", probe: config.LivenessProbe, start: true},
	} {
		if err := validateProbe(item.probe, item.name, item.start); err != nil {
			return err
		}
	}
	if err := validateVolumes(config.Volumes, config.VolumeMounts); err != nil {
		return err
	}
	if err := validateSecurityContext(config.SecurityContext); err != nil {
		return err
	}
	if config.TerminationGracePeriodSeconds != nil &&
		(*config.TerminationGracePeriodSeconds < 0 || *config.TerminationGracePeriodSeconds > 300) {
		return fmt.Errorf("终止宽限期必须是 0-300 秒")
	}
	seenSecrets := make(map[string]bool, len(config.ImagePullSecrets))
	for _, name := range config.ImagePullSecrets {
		if err := validateResourceName(name, "imagePullSecret"); err != nil {
			return err
		}
		if seenSecrets[name] {
			return fmt.Errorf("imagePullSecret 名称重复: %s", name)
		}
		seenSecrets[name] = true
	}
	return nil
}

func validateEnvironment(config *model.AppConfig) error {
	seenNames := make(map[string]bool, len(config.Env))
	for _, env := range config.Env {
		if messages := utilvalidation.IsEnvVarName(env.Name); len(messages) > 0 {
			return fmt.Errorf("环境变量名称 %q 无效: %s", env.Name, messages[0])
		}
		if seenNames[env.Name] {
			return fmt.Errorf("环境变量名称重复: %s", env.Name)
		}
		seenNames[env.Name] = true
		if (env.Value == nil) == (env.ValueFrom == nil) {
			return fmt.Errorf("环境变量 %s 必须且只能设置 value 或 value_from", env.Name)
		}
		if env.ValueFrom != nil {
			if (env.ValueFrom.ConfigMapKeyRef == nil) == (env.ValueFrom.SecretKeyRef == nil) {
				return fmt.Errorf("环境变量 %s 的 value_from 必须且只能设置一个引用", env.Name)
			}
			if env.ValueFrom.ConfigMapKeyRef != nil {
				if err := validateKeyReference(env.ValueFrom.ConfigMapKeyRef, "ConfigMap"); err != nil {
					return err
				}
			}
			if env.ValueFrom.SecretKeyRef != nil {
				if err := validateKeyReference(env.ValueFrom.SecretKeyRef, "Secret"); err != nil {
					return err
				}
			}
		}
	}
	seenEnvFrom := make(map[string]bool, len(config.EnvFrom))
	for _, source := range config.EnvFrom {
		if source.Prefix != "" {
			if messages := utilvalidation.IsCIdentifier(source.Prefix); len(messages) > 0 {
				return fmt.Errorf("环境变量前缀 %q 无效", source.Prefix)
			}
		}
		if (source.ConfigMapRef == nil) == (source.SecretRef == nil) {
			return fmt.Errorf("env_from 必须且只能设置一个引用")
		}
		if source.ConfigMapRef != nil {
			if err := validateResourceName(source.ConfigMapRef.Name, "ConfigMap"); err != nil {
				return err
			}
			key := "configmap/" + source.ConfigMapRef.Name + "/" + source.Prefix
			if seenEnvFrom[key] {
				return fmt.Errorf("env_from 引用重复: %s", source.ConfigMapRef.Name)
			}
			seenEnvFrom[key] = true
		}
		if source.SecretRef != nil {
			if err := validateResourceName(source.SecretRef.Name, "Secret"); err != nil {
				return err
			}
			key := "secret/" + source.SecretRef.Name + "/" + source.Prefix
			if seenEnvFrom[key] {
				return fmt.Errorf("env_from 引用重复: %s", source.SecretRef.Name)
			}
			seenEnvFrom[key] = true
		}
	}
	return nil
}

func validateKeyReference(reference *model.KeyReference, kind string) error {
	if err := validateResourceName(reference.Name, kind); err != nil {
		return err
	}
	if reference.Key == "" {
		return fmt.Errorf("%s 引用键不能为空", kind)
	}
	if messages := utilvalidation.IsConfigMapKey(reference.Key); len(messages) > 0 {
		return fmt.Errorf("%s 引用键 %q 无效: %s", kind, reference.Key, messages[0])
	}
	return nil
}

func validateResources(resources *model.ResourceRequirements) error {
	if resources == nil {
		return nil
	}
	requestCPU, err := validateQuantity(resources.Requests.CPU, "CPU request")
	if err != nil {
		return err
	}
	limitCPU, err := validateQuantity(resources.Limits.CPU, "CPU limit")
	if err != nil {
		return err
	}
	requestMemory, err := validateQuantity(resources.Requests.Memory, "内存 request")
	if err != nil {
		return err
	}
	limitMemory, err := validateQuantity(resources.Limits.Memory, "内存 limit")
	if err != nil {
		return err
	}
	if requestCPU != nil && limitCPU != nil && requestCPU.Cmp(*limitCPU) > 0 {
		return fmt.Errorf("CPU request 不能大于 limit")
	}
	if requestMemory != nil && limitMemory != nil && requestMemory.Cmp(*limitMemory) > 0 {
		return fmt.Errorf("内存 request 不能大于 limit")
	}
	return nil
}

func validateQuantity(value, label string) (*apiresource.Quantity, error) {
	if value == "" {
		return nil, nil
	}
	quantity, err := apiresource.ParseQuantity(value)
	if err != nil || quantity.Sign() <= 0 {
		return nil, fmt.Errorf("%s 必须是正数 Kubernetes Quantity", label)
	}
	return &quantity, nil
}

func validatePorts(ports []model.AppPort) error {
	names := make(map[string]bool, len(ports))
	containers := make(map[string]bool, len(ports))
	services := make(map[string]bool, len(ports))
	for index := range ports {
		port := &ports[index]
		if messages := utilvalidation.IsValidPortName(port.Name); len(messages) > 0 {
			return fmt.Errorf("端口名称 %q 无效: %s", port.Name, messages[0])
		}
		if names[port.Name] {
			return fmt.Errorf("端口名称重复: %s", port.Name)
		}
		names[port.Name] = true
		if port.ContainerPort < 1 || port.ContainerPort > 65535 {
			return fmt.Errorf("容器端口必须是 1-65535 的整数")
		}
		if port.Protocol == "" {
			port.Protocol = "TCP"
		}
		if port.Protocol != "TCP" && port.Protocol != "UDP" {
			return fmt.Errorf("端口协议仅支持 TCP 或 UDP")
		}
		containerKey := fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol)
		if containers[containerKey] {
			return fmt.Errorf("容器端口重复: %s", containerKey)
		}
		containers[containerKey] = true
		if port.ServicePort != nil {
			if *port.ServicePort < 1 || *port.ServicePort > 65535 {
				return fmt.Errorf("service 端口必须是 1-65535 的整数")
			}
			serviceKey := fmt.Sprintf("%d/%s", *port.ServicePort, port.Protocol)
			if services[serviceKey] {
				return fmt.Errorf("service 端口重复: %s", serviceKey)
			}
			services[serviceKey] = true
		}
	}
	return nil
}

func validateProbe(probe *model.Probe, label string, successMustBeOne bool) error {
	if probe == nil {
		return nil
	}
	handlers := 0
	if probe.HTTPGet != nil {
		handlers++
		if err := validateAbsolutePath(probe.HTTPGet.Path, label+" HTTP path", false); err != nil {
			return err
		}
		if probe.HTTPGet.Port < 1 || probe.HTTPGet.Port > 65535 {
			return fmt.Errorf("%s HTTP 端口无效", label)
		}
		if probe.HTTPGet.Scheme == "" {
			probe.HTTPGet.Scheme = "HTTP"
		}
		if probe.HTTPGet.Scheme != "HTTP" && probe.HTTPGet.Scheme != "HTTPS" {
			return fmt.Errorf("%s HTTP scheme 无效", label)
		}
		if len(probe.HTTPGet.HTTPHeaders) > 20 {
			return fmt.Errorf("%s HTTP header 最多 20 项", label)
		}
		for _, header := range probe.HTTPGet.HTTPHeaders {
			if !headerNamePattern.MatchString(header.Name) {
				return fmt.Errorf("%s HTTP header 名称无效", label)
			}
			if strings.ContainsAny(header.Value, "\r\n") {
				return fmt.Errorf("%s HTTP header 值无效", label)
			}
		}
	}
	if probe.TCPSocket != nil {
		handlers++
		if probe.TCPSocket.Port < 1 || probe.TCPSocket.Port > 65535 {
			return fmt.Errorf("%s TCP 端口无效", label)
		}
	}
	if probe.Exec != nil {
		handlers++
		if len(probe.Exec.Command) == 0 || len(probe.Exec.Command) > 20 {
			return fmt.Errorf("%s Exec command 必须包含 1-20 项", label)
		}
	}
	if handlers != 1 {
		return fmt.Errorf("%s 必须且只能设置一个 handler", label)
	}
	if probe.InitialDelaySeconds < 0 || probe.InitialDelaySeconds > 3600 ||
		probe.PeriodSeconds < 0 || probe.PeriodSeconds > 3600 ||
		probe.TimeoutSeconds < 0 || probe.TimeoutSeconds > 3600 ||
		probe.SuccessThreshold < 0 || probe.SuccessThreshold > 10 ||
		probe.FailureThreshold < 0 || probe.FailureThreshold > 60 {
		return fmt.Errorf("%s 时间或阈值超出范围", label)
	}
	if successMustBeOne && probe.SuccessThreshold > 1 {
		return fmt.Errorf("%s success_threshold 只能是 0 或 1", label)
	}
	return nil
}

func validateVolumes(volumes []model.Volume, mounts []model.VolumeMount) error {
	names := make(map[string]bool, len(volumes))
	for _, volume := range volumes {
		if messages := utilvalidation.IsDNS1123Label(volume.Name); len(messages) > 0 {
			return fmt.Errorf("卷名称 %q 无效: %s", volume.Name, messages[0])
		}
		if names[volume.Name] {
			return fmt.Errorf("卷名称重复: %s", volume.Name)
		}
		names[volume.Name] = true
		sources := 0
		if volume.EmptyDir != nil {
			sources++
			if volume.EmptyDir.Medium != "" && volume.EmptyDir.Medium != "Memory" {
				return fmt.Errorf("emptyDir medium 仅支持空值或 Memory")
			}
			if _, err := validateQuantity(volume.EmptyDir.SizeLimit, "emptyDir size_limit"); err != nil {
				return err
			}
		}
		if volume.PersistentVolumeClaim != nil {
			sources++
			if err := validateResourceName(volume.PersistentVolumeClaim.ClaimName, "PVC"); err != nil {
				return err
			}
		}
		if volume.ConfigMap != nil {
			sources++
			if err := validateResourceName(volume.ConfigMap.Name, "ConfigMap"); err != nil {
				return err
			}
		}
		if volume.Secret != nil {
			sources++
			if err := validateResourceName(volume.Secret.Name, "Secret"); err != nil {
				return err
			}
		}
		if sources != 1 {
			return fmt.Errorf("卷 %s 必须且只能设置一个来源", volume.Name)
		}
	}
	paths := make(map[string]bool, len(mounts))
	for _, mount := range mounts {
		if !names[mount.Name] {
			return fmt.Errorf("卷挂载引用了未声明的卷: %s", mount.Name)
		}
		if err := validateAbsolutePath(mount.MountPath, "卷挂载路径", false); err != nil {
			return err
		}
		if paths[mount.MountPath] {
			return fmt.Errorf("卷挂载路径重复: %s", mount.MountPath)
		}
		paths[mount.MountPath] = true
		if mount.SubPath != "" && (len(mount.SubPath) > maxPathLength || path.IsAbs(mount.SubPath) || hasParentPath(mount.SubPath)) {
			return fmt.Errorf("卷挂载 sub_path 必须是无 .. 的相对路径")
		}
	}
	return nil
}

func validateSecurityContext(security *model.SecurityContext) error {
	if security == nil {
		return nil
	}
	for _, id := range []*int64{security.RunAsUser, security.RunAsGroup, security.FSGroup} {
		if id != nil && (*id < 0 || *id > 4294967295) {
			return fmt.Errorf("安全上下文 ID 必须是 0-4294967295")
		}
	}
	if security.AllowPrivilegeEscalation != nil && *security.AllowPrivilegeEscalation {
		return fmt.Errorf("不允许提升容器权限")
	}
	if security.SeccompProfile != "" && security.SeccompProfile != "RuntimeDefault" {
		return fmt.Errorf("seccomp_profile 仅支持 RuntimeDefault")
	}
	if len(security.DropCapabilities) > 20 {
		return fmt.Errorf("drop capabilities 最多 20 项")
	}
	seen := make(map[string]bool, len(security.DropCapabilities))
	for index, capability := range security.DropCapabilities {
		capability = strings.ToUpper(strings.TrimSpace(capability))
		if !allowedCapabilities[capability] {
			return fmt.Errorf("capability 名称无效")
		}
		if seen[capability] {
			return fmt.Errorf("capability 名称重复: %s", capability)
		}
		seen[capability] = true
		security.DropCapabilities[index] = capability
	}
	return nil
}

func validateAbsolutePath(value, label string, allowEmpty bool) error {
	if value == "" && allowEmpty {
		return nil
	}
	if value == "" || len(value) > maxPathLength || !path.IsAbs(value) {
		return fmt.Errorf("%s 必须是长度不超过 %d 的绝对 POSIX 路径", label, maxPathLength)
	}
	return nil
}

func hasParentPath(value string) bool {
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func validateResourceName(name, label string) error {
	if messages := utilvalidation.IsDNS1123Subdomain(name); len(messages) > 0 {
		return fmt.Errorf("%s 名称 %q 无效: %s", label, name, messages[0])
	}
	return nil
}
