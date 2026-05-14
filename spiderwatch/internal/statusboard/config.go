// Package statusboard implements the public Spider Status Board web service.
// It is a read-only view of run results received from SpiderWatch.
package statusboard

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration for the Status Board.
type Config struct {
	Server ServerConfig `yaml:"server"`
	Auth   AuthConfig   `yaml:"auth"`
	Log    LogConfig    `yaml:"log"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
}

// AuthConfig holds the shared secret used to authenticate pushes from SpiderWatch.
type AuthConfig struct {
	Token string `yaml:"token"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

var (
	mu      sync.RWMutex
	current *Config
	cfgLog  = logrus.New()
)

// Load reads the YAML config file and starts a file watcher for hot-reload.
func Load(path string, onChange func(*Config)) (*Config, error) {
	cfg, err := readFile(path)
	if err != nil {
		return nil, err
	}
	applyDefaults(cfg)

	mu.Lock()
	current = cfg
	mu.Unlock()

	go watch(path, onChange)
	return cfg, nil
}

// Get returns the current configuration (thread-safe).
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func readFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}
	data = []byte(os.ExpandEnv(string(data)))
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 4096
	}
	if cfg.Server.Address == "" {
		cfg.Server.Address = "0.0.0.0"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
}

func watch(path string, onChange func(*Config)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cfgLog.WithError(err).Warn("statusboard config: cannot start file watcher")
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		cfgLog.WithError(err).Warnf("statusboard config: cannot watch %q", path)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				cfg, err := readFile(path)
				if err != nil {
					cfgLog.WithError(err).Warn("statusboard config: reload failed")
					continue
				}
				applyDefaults(cfg)
				mu.Lock()
				current = cfg
				mu.Unlock()
				cfgLog.Info("statusboard config: hot-reloaded")
				if onChange != nil {
					onChange(cfg)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			cfgLog.WithError(err).Warn("statusboard config: watcher error")
		}
	}
}

// SetupLogger configures the global logrus logger from config.
func SetupLogger(cfg *Config) *logrus.Logger {
	l := logrus.New()
	level, err := logrus.ParseLevel(strings.ToLower(cfg.Log.Level))
	if err != nil {
		level = logrus.InfoLevel
	}
	l.SetLevel(level)
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	if cfg.Log.File != "" {
		if err := os.MkdirAll(dirOf(cfg.Log.File), 0o755); err == nil {
			f, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err == nil {
				l.SetOutput(f)
			}
		}
	}
	return l
}

func dirOf(path string) string {
	idx := strings.LastIndexByte(path, '/')
	if idx < 0 {
		return "."
	}
	return path[:idx]
}
