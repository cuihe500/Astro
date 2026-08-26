package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// User 用户模型
type User struct {
	BaseModel
	UUID     string `gorm:"type:char(36);uniqueIndex;not null" json:"uuid"`
	Username string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password string `gorm:"size:128;not null" json:"-"`
	Email    string `gorm:"size:128;uniqueIndex" json:"email"`
	Status   int    `gorm:"default:1" json:"status"`
}

// BeforeCreate 创建用户前自动生成 UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.UUID = uuid.New().String()
	return nil
}

// OAuthIdentity OAuth2 身份模型
type OAuthIdentity struct {
	BaseModel
	Provider       string `gorm:"size:64;uniqueIndex:idx_oauth_provider_user;not null" json:"provider"`
	ProviderUserID string `gorm:"size:128;uniqueIndex:idx_oauth_provider_user;not null" json:"provider_user_id"`
	UserID         uint   `gorm:"index;not null" json:"user_id"`
}

// Project 用户项目模型
type Project struct {
	BaseModel
	Name      string `gorm:"size:64;not null;uniqueIndex:idx_projects_user_name" json:"name"`
	UserID    uint   `gorm:"not null;index;uniqueIndex:idx_projects_user_name" json:"user_id"`
	Namespace string `gorm:"size:63;not null;uniqueIndex" json:"namespace"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
}

// AppConfig 应用创建时保存的受控单容器配置。
type AppConfig struct {
	Command                       []string              `json:"command,omitempty"`
	Args                          []string              `json:"args,omitempty"`
	WorkingDir                    string                `json:"working_dir,omitempty"`
	ImagePullPolicy               string                `json:"image_pull_policy,omitempty"`
	Env                           []EnvVar              `json:"env,omitempty"`
	EnvFrom                       []EnvFromSource       `json:"env_from,omitempty"`
	Resources                     *ResourceRequirements `json:"resources,omitempty"`
	Ports                         []AppPort             `json:"ports,omitempty"`
	StartupProbe                  *Probe                `json:"startup_probe,omitempty"`
	ReadinessProbe                *Probe                `json:"readiness_probe,omitempty"`
	LivenessProbe                 *Probe                `json:"liveness_probe,omitempty"`
	Volumes                       []Volume              `json:"volumes,omitempty"`
	VolumeMounts                  []VolumeMount         `json:"volume_mounts,omitempty"`
	SecurityContext               *SecurityContext      `json:"security_context,omitempty"`
	TerminationGracePeriodSeconds *int64                `json:"termination_grace_period_seconds,omitempty"`
	ImagePullSecrets              []string              `json:"image_pull_secrets,omitempty"`
}

// EnvVar 容器环境变量。
type EnvVar struct {
	Name      string        `json:"name"`
	Value     *string       `json:"value,omitempty"`
	ValueFrom *EnvVarSource `json:"value_from,omitempty"`
}

// EnvVarSource 环境变量键引用。
type EnvVarSource struct {
	ConfigMapKeyRef *KeyReference `json:"config_map_key_ref,omitempty"`
	SecretKeyRef    *KeyReference `json:"secret_key_ref,omitempty"`
}

// KeyReference ConfigMap 或 Secret 键引用。
type KeyReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// EnvFromSource 整体导入的 ConfigMap 或 Secret 引用。
type EnvFromSource struct {
	Prefix       string               `json:"prefix,omitempty"`
	ConfigMapRef *NamedResourceSource `json:"config_map_ref,omitempty"`
	SecretRef    *NamedResourceSource `json:"secret_ref,omitempty"`
}

// NamedResourceSource 同命名空间资源引用。
type NamedResourceSource struct {
	Name string `json:"name"`
}

// ResourceRequirements 容器资源请求与限制。
type ResourceRequirements struct {
	Requests ResourceValues `json:"requests,omitempty"`
	Limits   ResourceValues `json:"limits,omitempty"`
}

// ResourceValues CPU 与内存资源值。
type ResourceValues struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// AppPort 容器端口及可选的 ClusterIP Service 端口。
type AppPort struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
	ServicePort   *int32 `json:"service_port,omitempty"`
}

// Probe 容器健康探针。
type Probe struct {
	HTTPGet             *HTTPGetAction   `json:"http_get,omitempty"`
	TCPSocket           *TCPSocketAction `json:"tcp_socket,omitempty"`
	Exec                *ExecAction      `json:"exec,omitempty"`
	InitialDelaySeconds int32            `json:"initial_delay_seconds,omitempty"`
	PeriodSeconds       int32            `json:"period_seconds,omitempty"`
	TimeoutSeconds      int32            `json:"timeout_seconds,omitempty"`
	SuccessThreshold    int32            `json:"success_threshold,omitempty"`
	FailureThreshold    int32            `json:"failure_threshold,omitempty"`
}

// HTTPGetAction HTTP 探针动作。
type HTTPGetAction struct {
	Path        string       `json:"path"`
	Port        int32        `json:"port"`
	Scheme      string       `json:"scheme,omitempty"`
	HTTPHeaders []HTTPHeader `json:"http_headers,omitempty"`
}

// HTTPHeader HTTP 探针请求头。
type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TCPSocketAction TCP 探针动作。
type TCPSocketAction struct {
	Port int32 `json:"port"`
}

// ExecAction Exec 探针动作。
type ExecAction struct {
	Command []string `json:"command"`
}

// Volume Pod 卷。
type Volume struct {
	Name                  string                       `json:"name"`
	EmptyDir              *EmptyDirVolumeSource        `json:"empty_dir,omitempty"`
	PersistentVolumeClaim *PersistentVolumeClaimSource `json:"persistent_volume_claim,omitempty"`
	ConfigMap             *NamedResourceSource         `json:"config_map,omitempty"`
	Secret                *NamedResourceSource         `json:"secret,omitempty"`
}

// EmptyDirVolumeSource emptyDir 卷配置。
type EmptyDirVolumeSource struct {
	Medium    string `json:"medium,omitempty"`
	SizeLimit string `json:"size_limit,omitempty"`
}

// PersistentVolumeClaimSource 已有 PVC 引用。
type PersistentVolumeClaimSource struct {
	ClaimName string `json:"claim_name"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

// VolumeMount 容器卷挂载。
type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SubPath   string `json:"sub_path,omitempty"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

// SecurityContext 受控的 Pod 与容器安全上下文。
type SecurityContext struct {
	RunAsNonRoot             *bool    `json:"run_as_non_root,omitempty"`
	RunAsUser                *int64   `json:"run_as_user,omitempty"`
	RunAsGroup               *int64   `json:"run_as_group,omitempty"`
	FSGroup                  *int64   `json:"fs_group,omitempty"`
	ReadOnlyRootFilesystem   *bool    `json:"read_only_root_filesystem,omitempty"`
	AllowPrivilegeEscalation *bool    `json:"allow_privilege_escalation,omitempty"`
	DropCapabilities         []string `json:"drop_capabilities,omitempty"`
	SeccompProfile           string   `json:"seccomp_profile,omitempty"`
}

// App 应用模型
type App struct {
	BaseModel
	Name      string    `gorm:"size:64;not null;uniqueIndex:idx_apps_project_name" json:"name"`
	Image     string    `gorm:"size:256;not null" json:"image"`
	Replicas  int       `gorm:"default:1" json:"replicas"`
	Status    string    `gorm:"size:32;default:stopped" json:"status"`
	Config    AppConfig `gorm:"serializer:json;type:json" json:"-"`
	ProjectID uint      `gorm:"not null;index;uniqueIndex:idx_apps_project_name" json:"project_id"`
	Project   Project   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
}
