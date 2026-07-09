// Package sshconf parses ~/.ssh/config and exposes the Host entries defined
// there for interactive selection.
package sshconf

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// Host is one selectable SSH destination, resolved from a Host block in
// ~/.ssh/config (or constructed manually by the caller for a one-off entry).
type Host struct {
	Name     string // the alias as it appears in the Host line
	Hostname string // HostName, or Name if unset
	User     string
	Port     int
}

// Config holds the Host entries parsed from an ssh_config file.
type Config struct {
	hosts []Host
}

// LoadConfig parses the current user's ~/.ssh/config. A missing file is not
// an error: it yields an empty Config, since manual host entry is always
// available regardless.
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("sshconf: load: %w", err)
	}
	return LoadConfigFile(filepath.Join(home, ".ssh", "config"))
}

// LoadConfigFile parses the ssh_config file at path.
func LoadConfigFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("sshconf: load %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("sshconf: load %q: %w", path, err)
	}

	var hosts []Host
	seen := make(map[string]bool)
	for _, block := range cfg.Hosts {
		for _, pattern := range block.Patterns {
			name := pattern.String()
			if strings.ContainsAny(name, "*?") || seen[name] {
				continue
			}
			seen[name] = true
			host, err := resolveHost(cfg, name)
			if err != nil {
				return nil, fmt.Errorf("sshconf: load %q: %w", path, err)
			}
			hosts = append(hosts, host)
		}
	}
	return &Config{hosts: hosts}, nil
}

func resolveHost(cfg *ssh_config.Config, alias string) (Host, error) {
	host := Host{Name: alias, Hostname: alias, Port: 22}

	if v, err := cfg.Get(alias, "HostName"); err != nil {
		return Host{}, err
	} else if v != "" {
		host.Hostname = v
	}

	if v, err := cfg.Get(alias, "User"); err != nil {
		return Host{}, err
	} else if v != "" {
		host.User = v
	}

	if v, err := cfg.Get(alias, "Port"); err != nil {
		return Host{}, err
	} else if v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return Host{}, fmt.Errorf("host %q: invalid Port %q: %w", alias, v, err)
		}
		host.Port = port
	}

	return host, nil
}

// Hosts returns all Host entries parsed from the config file, in the order
// they appeared.
func (c *Config) Hosts() []Host {
	return c.hosts
}

// FindHost returns the Host with the given alias, or nil if not present.
func (c *Config) FindHost(name string) *Host {
	for i := range c.hosts {
		if c.hosts[i].Name == name {
			return &c.hosts[i]
		}
	}
	return nil
}
