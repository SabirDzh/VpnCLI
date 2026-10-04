package shared

import (
	"context"
	"errors"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// DescribeError maps sentinel errors to human-readable TUI text.
func DescribeError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, domain.ErrNotPrivileged):
		return "Нужен root для TUN. Запусти: sudo vpn tui"
	case errors.Is(err, domain.ErrNoActiveProfile):
		return "Профиль не выбран. Перейди на вкладку Profiles и нажми enter"
	case errors.Is(err, domain.ErrProfileNotFound):
		return "Профиль не найден. Обнови список (r)"
	case errors.Is(err, domain.ErrSubscriptionNotFound):
		return "Подписка не найдена. Обнови список (r)"
	case errors.Is(err, domain.ErrCoreNotFound):
		return "Ядро не найдено. Проверь sing-box в PATH"
	case errors.Is(err, domain.ErrUnsupportedProtocol):
		return "Протокол не поддерживается активным ядром"
	case errors.Is(err, domain.ErrAlreadyRunning):
		return "VPN уже подключён"
	case errors.Is(err, domain.ErrNotRunning):
		return "VPN уже отключён"
	case errors.Is(err, context.DeadlineExceeded):
		return "Превышено время ожидания. Повтори операцию"
	default:
		return "Ошибка: " + err.Error()
	}
}
