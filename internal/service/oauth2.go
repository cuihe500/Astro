package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/internal/repository"
	"github.com/cuihe500/astro/pkg/config"
	"github.com/cuihe500/astro/pkg/errcode"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	oauth2StateTTL = 10 * time.Minute

	bytCloudAuthAlias       = "bytcloudauth"
	bytCloudAuthProviderKey = "authentik"
)

// OAuth2Service OAuth2 登录服务
type OAuth2Service struct {
	repo    *repository.UserRepository
	userSvc *UserService
}

// NewOAuth2Service 创建 OAuth2 登录服务
func NewOAuth2Service() *OAuth2Service {
	return &OAuth2Service{
		repo:    repository.NewUserRepository(),
		userSvc: NewUserService(),
	}
}

// BuildAuthURL 生成 OAuth2 授权 URL
func (s *OAuth2Service) BuildAuthURL(providerAlias string) (string, error) {
	providerKey, err := resolveOAuth2Provider(providerAlias)
	if err != nil {
		return "", err
	}

	providerConfig, err := s.getProviderConfig(providerKey)
	if err != nil {
		return "", err
	}

	state, err := generateOAuth2State(providerAlias, config.GlobalConfig.JWT.Secret, time.Now())
	if err != nil {
		return "", errcode.New(errcode.ErrInternal)
	}

	return newOAuth2Config(providerConfig).AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

// Callback 处理 OAuth2 回调并签发 JWT
func (s *OAuth2Service) Callback(ctx context.Context, providerAlias, code, state string) (string, *model.User, error) {
	providerKey, err := resolveOAuth2Provider(providerAlias)
	if err != nil {
		return "", nil, err
	}

	providerConfig, err := s.getProviderConfig(providerKey)
	if err != nil {
		return "", nil, err
	}
	if err := validateOAuth2State(state, providerAlias, config.GlobalConfig.JWT.Secret, time.Now()); err != nil {
		return "", nil, errcode.New(errcode.ErrUnauthorized)
	}

	token, err := newOAuth2Config(providerConfig).Exchange(ctx, code)
	if err != nil {
		return "", nil, errcode.New(errcode.ErrLoginFailed)
	}

	userInfo, err := fetchOAuth2UserInfo(ctx, providerConfig.UserInfoURL, token)
	if err != nil {
		return "", nil, err
	}

	user, err := s.findOrCreateUser(providerKey, userInfo)
	if err != nil {
		return "", nil, err
	}

	jwtToken, err := s.userSvc.generateToken(user.ID, user.UUID)
	if err != nil {
		return "", nil, errcode.New(errcode.ErrInternal)
	}
	return jwtToken, user, nil
}

func resolveOAuth2Provider(providerAlias string) (string, error) {
	if providerAlias != bytCloudAuthAlias {
		return "", errcode.NewWithMsg(errcode.ErrBadRequest, "OAuth2 Provider 不可用")
	}
	return bytCloudAuthProviderKey, nil
}

// newOAuth2DatabaseError 返回不暴露内部 Provider 键的数据库错误。
func newOAuth2DatabaseError() *errcode.Error {
	return errcode.New(errcode.ErrDatabase)
}

func (s *OAuth2Service) getProviderConfig(providerKey string) (config.OAuth2ProviderConfig, error) {
	if config.GlobalConfig == nil {
		return config.OAuth2ProviderConfig{}, errcode.New(errcode.ErrInternal)
	}
	providerConfig, ok := config.GlobalConfig.OAuth2.Providers[providerKey]
	if !ok || !providerConfig.Enabled || providerConfig.ClientID == "" || providerConfig.ClientSecret == "" || providerConfig.RedirectURL == "" || providerConfig.AuthURL == "" || providerConfig.TokenURL == "" || providerConfig.UserInfoURL == "" {
		return config.OAuth2ProviderConfig{}, errcode.NewWithMsg(errcode.ErrBadRequest, "OAuth2 Provider 不可用")
	}
	return providerConfig, nil
}

func (s *OAuth2Service) findOrCreateUser(provider string, userInfo oauth2UserInfo) (*model.User, error) {
	providerUserID := userInfo.providerUserID()
	if providerUserID == "" {
		return nil, errcode.New(errcode.ErrLoginFailed)
	}

	identity, err := s.repo.GetOAuthIdentity(provider, providerUserID)
	if err == nil {
		return s.getUserByID(identity.UserID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, newOAuth2DatabaseError()
	}

	if userInfo.Email != "" {
		_, err := s.repo.GetUserByEmail(userInfo.Email)
		if err == nil {
			return nil, errcode.New(errcode.ErrEmailExists)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newOAuth2DatabaseError()
		}
	}

	username, err := s.uniqueUsername(userInfo.usernameSource(), provider, providerUserID)
	if err != nil {
		return nil, err
	}
	email := userInfo.Email
	if email == "" {
		email = fmt.Sprintf("%s-%s@oauth.local", provider, shortHash(providerUserID))
	}
	// Password 使用非 bcrypt 占位值，确保 OAuth2 用户不能通过本地密码登录。
	user := &model.User{Username: username, Password: "oauth2", Email: email}
	identity = &model.OAuthIdentity{Provider: provider, ProviderUserID: providerUserID}
	if err := s.repo.CreateOAuthUser(user, identity); err != nil {
		return nil, newOAuth2DatabaseError()
	}
	return user, nil
}

func (s *OAuth2Service) getUserByID(id uint) (*model.User, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.New(errcode.ErrUserNotFound)
		}
		return nil, newOAuth2DatabaseError()
	}
	return user, nil
}

func (s *OAuth2Service) uniqueUsername(source, provider, providerUserID string) (string, error) {
	base := normalizeUsername(source)
	if base == "" {
		base = "oauth"
	}
	suffix := shortHash(provider + ":" + providerUserID)
	for attempt := 0; attempt < 10; attempt++ {
		username := base
		if attempt > 0 {
			username = withUsernameSuffix(base, fmt.Sprintf("%s%d", suffix, attempt))
		}
		_, err := s.repo.GetUserByUsername(username)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return username, nil
		}
		if err != nil {
			return "", newOAuth2DatabaseError()
		}
	}
	return "", errcode.New(errcode.ErrRegisterFailed)
}

type oauth2UserInfo struct {
	Sub               string `json:"sub"`
	ID                string `json:"id"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
}

func (u oauth2UserInfo) providerUserID() string {
	if u.Sub != "" {
		return u.Sub
	}
	return u.ID
}

func (u oauth2UserInfo) usernameSource() string {
	if u.PreferredUsername != "" {
		return u.PreferredUsername
	}
	if u.Name != "" {
		return u.Name
	}
	if u.Email != "" {
		local, _, found := strings.Cut(u.Email, "@")
		if found {
			return local
		}
		return u.Email
	}
	return ""
}

func fetchOAuth2UserInfo(ctx context.Context, userInfoURL string, token *oauth2.Token) (userInfo oauth2UserInfo, resultErr error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return oauth2UserInfo{}, errcode.New(errcode.ErrLoginFailed)
	}
	token.SetAuthHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return oauth2UserInfo{}, errcode.New(errcode.ErrLoginFailed)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && resultErr == nil {
			resultErr = errcode.New(errcode.ErrLoginFailed)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return oauth2UserInfo{}, errcode.New(errcode.ErrLoginFailed)
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return oauth2UserInfo{}, errcode.New(errcode.ErrLoginFailed)
	}
	return userInfo, nil
}

func newOAuth2Config(providerConfig config.OAuth2ProviderConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     providerConfig.ClientID,
		ClientSecret: providerConfig.ClientSecret,
		RedirectURL:  providerConfig.RedirectURL,
		Scopes:       providerConfig.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  providerConfig.AuthURL,
			TokenURL: providerConfig.TokenURL,
		},
	}
}

type oauth2StatePayload struct {
	Provider  string `json:"provider"`
	ExpiresAt int64  `json:"exp"`
	Nonce     string `json:"nonce"`
}

func generateOAuth2State(provider, secret string, now time.Time) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	payload := oauth2StatePayload{
		Provider:  provider,
		ExpiresAt: now.Add(oauth2StateTTL).Unix(),
		Nonce:     base64.RawURLEncoding.EncodeToString(nonce),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signaturePart, err := signOAuth2State(payloadPart, secret)
	if err != nil {
		return "", err
	}
	return payloadPart + "." + signaturePart, nil
}

func validateOAuth2State(state, provider, secret string, now time.Time) error {
	payloadPart, signaturePart, ok := strings.Cut(state, ".")
	if !ok || payloadPart == "" || signaturePart == "" {
		return errors.New("invalid state")
	}
	expectedSignature, err := signOAuth2State(payloadPart, secret)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(signaturePart), []byte(expectedSignature)) {
		return errors.New("invalid state")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return err
	}
	var payload oauth2StatePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return err
	}
	if payload.Provider != provider || payload.ExpiresAt < now.Unix() {
		return errors.New("invalid state")
	}
	return nil
}

func signOAuth2State(payloadPart, secret string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secret))
	written, err := mac.Write([]byte(payloadPart))
	if err != nil {
		return "", err
	}
	if written != len(payloadPart) {
		return "", errors.New("state 签名写入不完整")
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func normalizeUsername(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	var builder strings.Builder
	lastDash := false
	for _, r := range source {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-'
		if allowed {
			builder.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
		if builder.Len() >= 48 {
			break
		}
	}
	return strings.Trim(builder.String(), "-.")
}

func withUsernameSuffix(base, suffix string) string {
	maxBaseLen := 64 - len(suffix) - 1
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}
	return base + "-" + suffix
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}
