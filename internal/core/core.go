// Package core defines engine-neutral contracts. All sing-box knowledge
// lives in subpackages; the rest of the code depends only on this file.
package core

import (
	"context"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// Options — параметры генерации конфига, общие для всех ядер.
type Options struct {
	TUNEnabled   bool
	TUNName      string
	MTU          int
	Stack        string
	AutoRoute    bool
	StrictRoute  bool
	MixedEnabled bool
	MixedPort    int
	LogLevel     string
	// Features applied by the config builder (see singbox.Builder).
	Adblock      bool
	TrackerBlock bool
	SocialBlock  bool
	AppFirewall  []string
	SplitExclude []string
	SplitInclude []string
}

// StartRequest — всё, что нужно ядру для запуска.
type StartRequest struct {
	Profile    domain.Profile
	Options    Options
	RuntimeDir string
	LogFile    string
}

// RunInfo — результат запуска процессного ядра.
type RunInfo struct {
	PID        int
	ConfigPath string
	Core       string
}

// Status — состояние запущенного ядра.
type Status struct {
	Running   bool
	PID       int
	Core      string
	ProfileID string
	Since     time.Time
}

// Core — интерфейс ядра. Процессные ядра (sing-box, xray) запускаются
// через process.Supervisor; AmneziaWG позже реализует его по-своему.
type Core interface {
	Name() string
	Supports(p domain.Profile) bool
	Start(ctx context.Context, req StartRequest) (RunInfo, error)
	Stop(ctx context.Context, info RunInfo) error
	Status(ctx context.Context, info RunInfo) (Status, error)
}

// ConfigBuilder переводит нейтральный профиль в нативный конфиг ядра.
type ConfigBuilder interface {
	Build(p domain.Profile, opts Options) ([]byte, error)
}

// Settings — настройки ядра из config.yaml.
type Settings struct {
	BinaryPath string
	MinVersion string
}

// Factory создает Core из настроек. Регистрируется в main.go.
type Factory func(cfg Settings) (Core, error)
