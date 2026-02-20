package cli

import (
	"testing"
)

func TestGetServerURL(t *testing.T) {
	// Reset to defaults
	host = "localhost"
	port = 8080

	url := GetServerURL()
	expected := "http://localhost:8080"

	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestGetServerURL_CustomHostPort(t *testing.T) {
	host = "192.168.1.100"
	port = 9000

	url := GetServerURL()
	expected := "http://192.168.1.100:9000"

	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}

	// Reset
	host = "localhost"
	port = 8080
}

func TestIsJSON(t *testing.T) {
	jsonOut = false
	if IsJSON() {
		t.Error("expected false")
	}

	jsonOut = true
	if !IsJSON() {
		t.Error("expected true")
	}

	// Reset
	jsonOut = false
}

func TestIsVerbose(t *testing.T) {
	verbose = false
	if IsVerbose() {
		t.Error("expected false")
	}

	verbose = true
	if !IsVerbose() {
		t.Error("expected true")
	}

	// Reset
	verbose = false
}

func TestGetAuth(t *testing.T) {
	user = ""
	password = ""

	u, p := GetAuth()
	if u != "" || p != "" {
		t.Errorf("expected empty auth, got %s:%s", u, p)
	}

	user = "admin"
	password = "secret"

	u, p = GetAuth()
	if u != "admin" || p != "secret" {
		t.Errorf("expected admin:secret, got %s:%s", u, p)
	}

	// Reset
	user = ""
	password = ""
}

func TestGetConfigFile(t *testing.T) {
	cfgFile = ""
	if GetConfigFile() != "" {
		t.Error("expected empty config file")
	}

	cfgFile = "/path/to/config.yaml"
	if GetConfigFile() != "/path/to/config.yaml" {
		t.Errorf("expected /path/to/config.yaml, got %s", GetConfigFile())
	}

	// Reset
	cfgFile = ""
}

func TestSetVersion(t *testing.T) {
	SetVersion("1.2.3")

	if Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", Version)
	}

	// Reset
	Version = "0.2.0"
}

func TestNewClient(t *testing.T) {
	host = "localhost"
	port = 8080

	client := NewClient()

	if client == nil {
		t.Fatal("expected client, got nil")
	}

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected http://localhost:8080, got %s", client.baseURL)
	}
}

func TestNewClient_WithAuth(t *testing.T) {
	user = "admin"
	password = "secret"

	client := NewClient()

	if client.user != "admin" {
		t.Errorf("expected user admin, got %s", client.user)
	}

	if client.password != "secret" {
		t.Errorf("expected password secret, got %s", client.password)
	}

	// Reset
	user = ""
	password = ""
}
