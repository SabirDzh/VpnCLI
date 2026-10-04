//go:build darwin

package platform

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

// killswitchAnchor is the pf anchor holding the kill switch rules.
const killswitchAnchor = "vpn-killswitch"

// killswitchRulesPath is where the anchor rule file lives.
const killswitchRulesPath = "/etc/pf.anchors/vpn-killswitch"

// KillSwitchRules renders the pf rules that block all traffic except the
// loopback, VPN tunnels (utun*/tun*), private networks and DHCP. Loaded
// into their own anchor, they cut internet whenever the VPN is down and
// stay transparent while TUN carries the traffic.
func KillSwitchRules() string {
	return `set skip on lo0
block drop all
pass on { utun+ tun+ } keep state
pass to { 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, 224.0.0.0/4, ff00::/8, fc00::/7, fe80::/7 }
pass out udp proto udp to port { 67, 68 }
`
}

// KillSwitchRulesFor adds a pass rule for the VPN server's resolved
// addresses and port: sing-box's own uplink leaves via the physical
// interface, so without it the anchor would strangle the tunnel itself.
// Unresolvable hosts fail closed.
func KillSwitchRulesFor(serverHost string, serverPort int) (string, error) {
	if serverPort <= 0 || serverPort > 65535 {
		return "", fmt.Errorf("bad server port %d", serverPort)
	}
	ips, err := net.LookupHost(serverHost)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("resolve vpn endpoint %q: %w", serverHost, err)
	}
	var safe []string
	for _, ip := range ips {
		if net.ParseIP(ip) != nil {
			safe = append(safe, ip)
		}
	}
	if len(safe) == 0 {
		return "", fmt.Errorf("vpn endpoint %q resolved to no valid IPs", serverHost)
	}
	return KillSwitchRules() + fmt.Sprintf("pass out to { %s } port %d\n", strings.Join(safe, ", "), serverPort), nil
}

// EnableKillSwitch installs the anchor and turns pf on.
func EnableKillSwitch(serverHost string, serverPort int) error {
	rules, err := KillSwitchRulesFor(serverHost, serverPort)
	if err != nil {
		return err
	}
	if err := os.WriteFile(killswitchRulesPath, []byte(rules), 0o644); err != nil {
		return fmt.Errorf("write anchor rules: %w", err)
	}
	if out, err := exec.Command("pfctl", "-a", killswitchAnchor, "-f", killswitchRulesPath).CombinedOutput(); err != nil {
		return fmt.Errorf("pfctl load anchor: %w\n%s", err, out)
	}
	// -E enables pf and remembers that we did (pfctl -D reverts on disable).
	if out, err := exec.Command("pfctl", "-E").CombinedOutput(); err != nil {
		return fmt.Errorf("pfctl enable: %w\n%s", err, out)
	}
	return nil
}

// DisableKillSwitch flushes the anchor rules and turns pf back off if we
// were the ones who enabled it.
func DisableKillSwitch() error {
	if out, err := exec.Command("pfctl", "-a", killswitchAnchor, "-F", "all").CombinedOutput(); err != nil {
		return fmt.Errorf("pfctl flush anchor: %w\n%s", err, out)
	}
	// Best effort: -D only succeeds when a previous -E is ours; other
	// failures mean someone else manages pf, which we must not disturb.
	_, _ = exec.Command("pfctl", "-D").CombinedOutput()
	_ = os.Remove(killswitchRulesPath)
	return nil
}

// KillSwitchActive reports whether the anchor currently holds rules.
func KillSwitchActive() bool {
	out, err := exec.Command("pfctl", "-a", killswitchAnchor, "-s", "Rules").CombinedOutput()
	return err == nil && len(out) > 0
}
