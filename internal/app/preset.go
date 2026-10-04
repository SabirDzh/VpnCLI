package app

// RFApps lists process names of apps that must bypass the VPN in Russia
// (banks, government, ride-hailing). Both latin and cyrillic name
// variants are listed because macOS shows either depending on locale.
var RFApps = []string{
	"Госуслуги", "gosuslugi",
	"СберБанк", "СБОЛ", "SberBank", "СберБанк Онлайн",
	"Т-Банк", "Тинькофф", "Tinkoff", "TBank",
	"ВТБ", "VTB", "VTB24",
	"Альфа-Банк", "AlfaBank", "Альфа Онлайн",
	"Райффайзен", "Raiffeisen",
	"Яндекс", "Yandex", "Яндекс Музыка", "Yandex Music", "ЯндексБраузер",
	"VK", "VK Messenger", "ВКонтакте",
	"1С", "1cv8",
	"Делимобиль", "Delimobil", "Ситидрайв", "Citydrive",
}

// EffectiveApps merges base with the preset, preserving order and
// deduplicating (first occurrence wins).
func EffectiveApps(base []string, preset bool) []string {
	if !preset {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(RFApps))
	out := make([]string, 0, len(base)+len(RFApps))
	add := func(list []string) {
		for _, a := range list {
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			out = append(out, a)
		}
	}
	add(base)
	add(RFApps)
	return out
}
