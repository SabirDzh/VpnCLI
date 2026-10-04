# vpn — CLI VPN client (sing-box core)

MVP: sing-box ≥ 1.14, Linux/macOS, протоколы vless (incl. Reality),
vmess, trojan, shadowsocks, hysteria2. Режим TUN основной + локальный mixed-прокси.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/SabirDzh/VpnCLI/master/install.sh | sh
# конкретная версия:  VERSION=v0.1.0 curl ... | sh
# другой префикс:     PREFIX=~/.local curl ... | sh
# без ядра:           SKIP_SINGBOX=1 curl ... | sh
```

Скрипт ставит и сам CLI, и ядро sing-box (≥ 1.14):
macOS через Homebrew, Debian/Ubuntu через репозиторий `deb.sagernet.org`.
Если пакетного менеджера нет — скрипт скажет, что поставить вручную.

Windows (PowerShell, для TUN запускать как Administrator):

```powershell
irm https://raw.githubusercontent.com/SabirDzh/VpnCLI/master/install.ps1 | iex
```

Скрипт ставит и сам CLI, и ядро sing-box (≥ 1.14): сначала пробует
winget, затем choco и scoop, в крайнем случае качает sing-box и wintun
напрямую с апстрима. Для TUN нужны права администратора и `wintun.dll`
рядом с `sing-box.exe` (установщик кладет сам).

Требования: sing-box ≥ 1.14 в PATH, для TUN — root (Linux/macOS) или
Administrator (Windows, плюс wintun.dll рядом с sing-box).

## Build from source

```sh
# sing-box >= 1.14 must be installed
# https://sing-box.sagernet.org/installation/package-manager/
go build -o vpn ./cmd/vpn
```

## Quick start (smoke)

```sh
vpn sub add mysub "https://example.com/sub.txt"
vpn sub update
vpn profile list
vpn profile use <id|name>

sudo vpn up        # TUN, требует root
vpn status
curl -s ifconfig.me  # внешний IP должен измениться
sudo vpn down
```

Ручное добавление без подписки:

```sh
vpn profile add "vless://uuid@host:443?security=reality&sni=example.com&fp=chrome&pbk=KEY&sid=SID&flow=xtls-rprx-vision#name"
```

## Commands

```
vpn up [profile]            connect (default: active profile)
vpn down                    disconnect
vpn status                  connection status
vpn tui                     interactive terminal UI (needs sudo for TUN)
vpn update [--check]        check for and install CLI updates
vpn profile add|list|use|remove
vpn sub add|list|update|remove
vpn version
```

Global flags: `--config`, `--core`, `--log-level`, `--singbox-path`.
Env override: `VPN_*` (e.g. `VPN_LOG_LEVEL=debug`).

## Layout

- root → system paths (`/etc/vpn`, `/var/lib/vpn`, `/run/vpn`,
  `/var/log/vpn`); unprivileged → `~/.config/vpn`, `~/.local/share/vpn`.
- Generated sing-box config: `<runtime>/sing-box.json` (0600).
- State: `state.json` (core, profile, pid, started-at).

## Config file

`config.yaml` (viper: flags → `VPN_*` env → file → defaults):

```yaml
core:
  default: sing-box
  singbox:
    path: ""        # empty = PATH lookup
    min_version: "1.14.0"
log:
  level: info
tun:
  enabled: true
  mtu: 9000
  stack: system
  auto_route: true
  strict_route: true
mixed_port: 10808
```

## TUI

```sh
sudo vpn tui
```

Интерактивный интерфейс: статус, профили, подписки. TUN требует root,
поэтому без sudo TUI стартует в режиме чтения (статус и списки работают,
`up`/`down` объясняют, что нужен root). Выход из TUI (`q`, `ctrl+c`)
**не выключает VPN** — ядро живёт отдельным процессом.

| Клавиша | Действие |
|---|---|
| `1 2 3`, `tab` / `shift+tab` | переключение экранов |
| `↑ ↓` / `j k` | навигация |
| `enter` | основное действие экрана |
| `c` | подключить / отключить |
| `u` / `U` | обновить подписку / все подписки |
| `/` | фильтр (Profiles) |
| `r` | обновить данные |
| `?` | полная справка |
| `q`, `ctrl+c` | выход (VPN остаётся включённым) |

## Dev

```sh
make build test vet lint
make golden            # regenerate testdata/singbox/*.golden
make integration       # needs real sing-box (tag: integration)
```
