package sshconf

import (
	"path/filepath"
	"testing"
)

func TestLoadConfigFile(t *testing.T) {
	cfg, err := LoadConfigFile(filepath.Join("..", "..", "testdata", "ssh_config_sample"))
	if err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}

	hosts := cfg.Hosts()
	if len(hosts) != 3 {
		t.Fatalf("got %d hosts, want 3: %+v", len(hosts), hosts)
	}

	prod := cfg.FindHost("prod-db")
	if prod == nil {
		t.Fatal("prod-db not found")
	}
	want := Host{Name: "prod-db", Hostname: "10.0.1.5", User: "deploy", Port: 2222}
	if *prod != want {
		t.Errorf("prod-db = %+v, want %+v", *prod, want)
	}

	staging := cfg.FindHost("staging")
	if staging == nil {
		t.Fatal("staging not found")
	}
	if staging.Hostname != "staging.example.com" || staging.User != "ubuntu" || staging.Port != 22 {
		t.Errorf("staging = %+v, want HostName=staging.example.com User=ubuntu Port=22", *staging)
	}

	noHostname := cfg.FindHost("no-hostname")
	if noHostname == nil {
		t.Fatal("no-hostname not found")
	}
	if noHostname.Hostname != "no-hostname" {
		t.Errorf("no-hostname.Hostname = %q, want fallback to alias %q", noHostname.Hostname, "no-hostname")
	}

	// The wildcard "Host *" block must not appear as a selectable entry.
	if cfg.FindHost("*") != nil {
		t.Error("wildcard pattern leaked into Hosts()")
	}
}

func TestLoadConfigFileMissing(t *testing.T) {
	cfg, err := LoadConfigFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("LoadConfigFile on missing file: %v", err)
	}
	if len(cfg.Hosts()) != 0 {
		t.Errorf("expected empty Config, got %+v", cfg.Hosts())
	}
}

func TestLoadConfigFileBadPort(t *testing.T) {
	_, err := LoadConfigFile(filepath.Join("..", "..", "testdata", "ssh_config_bad_port"))
	if err == nil {
		t.Fatal("expected error for invalid Port value")
	}
}

func TestFindHostNotFound(t *testing.T) {
	cfg, err := LoadConfigFile(filepath.Join("..", "..", "testdata", "ssh_config_sample"))
	if err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	if cfg.FindHost("nonexistent") != nil {
		t.Error("expected nil for nonexistent host")
	}
}
