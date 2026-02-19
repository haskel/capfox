package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubstituteEnvVars(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	input := []byte("value: ${TEST_VAR}")
	expected := []byte("value: test_value")

	result := substituteEnvVars(input)

	if string(result) != string(expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSubstituteEnvVarsMultiple(t *testing.T) {
	os.Setenv("VAR1", "value1")
	os.Setenv("VAR2", "value2")
	defer os.Unsetenv("VAR1")
	defer os.Unsetenv("VAR2")

	input := []byte("first: ${VAR1}\nsecond: ${VAR2}")
	expected := []byte("first: value1\nsecond: value2")

	result := substituteEnvVars(input)

	if string(result) != string(expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSubstituteEnvVarsNotSet(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR")

	input := []byte("value: ${NONEXISTENT_VAR}")
	expected := []byte("value: ${NONEXISTENT_VAR}") // unchanged

	result := substituteEnvVars(input)

	if string(result) != string(expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSubstituteEnvVarsNoVars(t *testing.T) {
	input := []byte("value: plain_text")
	expected := []byte("value: plain_text")

	result := substituteEnvVars(input)

	if string(result) != string(expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestLoadWithEnvSubstitution(t *testing.T) {
	os.Setenv("TEST_HOST", "192.168.1.1")
	os.Setenv("TEST_PORT", "9999")
	defer os.Unsetenv("TEST_HOST")
	defer os.Unsetenv("TEST_PORT")

	content := `
server:
  host: "${TEST_HOST}"
  port: 9999

logging:
  level: "info"
  format: "json"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("expected host 192.168.1.1, got %s", cfg.Server.Host)
	}
}
