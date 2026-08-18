package service

import (
	"strings"
	"testing"
	"time"
)

func TestOAuth2State(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	state, err := generateOAuth2State("authentik", "secret", now)
	if err != nil {
		t.Fatalf("生成 state 失败: %v", err)
	}

	if err := validateOAuth2State(state, "authentik", "secret", now.Add(time.Minute)); err != nil {
		t.Fatalf("校验 state 失败: %v", err)
	}

	if err := validateOAuth2State(state+"x", "authentik", "secret", now); err == nil {
		t.Fatal("篡改 state 应该失败")
	}

	if err := validateOAuth2State(state, "other", "secret", now); err == nil {
		t.Fatal("provider 不匹配应该失败")
	}

	if err := validateOAuth2State(state, "authentik", "secret", now.Add(oauth2StateTTL+time.Second)); err == nil {
		t.Fatal("过期 state 应该失败")
	}
}

func TestOAuth2StateShape(t *testing.T) {
	state, err := generateOAuth2State("authentik", "secret", time.Now())
	if err != nil {
		t.Fatalf("生成 state 失败: %v", err)
	}
	if parts := strings.Split(state, "."); len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Fatalf("state 格式错误: %q", state)
	}
}
