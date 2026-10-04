package platform

import (
	"strings"
	"testing"
)

func TestKillSwitchRulesContent(t *testing.T) {
	rules := KillSwitchRules()
	for _, want := range []string{"set skip on lo0", "block drop all", "pass on { utun+ tun+ }", "192.168.0.0/16", "port { 67, 68 }"} {
		if !strings.Contains(rules, want) {
			t.Fatalf("rules missing %q:\n%s", want, rules)
		}
	}
}

func TestKillSwitchRulesForEndpoint(t *testing.T) {
	rules, err := KillSwitchRulesFor("localhost", 8443)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rules, "pass out to { ") || !strings.Contains(rules, "127.0.0.1") {
		t.Fatalf("endpoint rule missing:\n%s", rules)
	}
	if !strings.Contains(rules, "port 8443") {
		t.Fatalf("endpoint port missing:\n%s", rules)
	}
	if _, err := KillSwitchRulesFor("nonexistent-host-for-test.invalid", 443); err == nil {
		t.Fatal("unresolvable host must fail closed")
	}
	if _, err := KillSwitchRulesFor("example.com", 0); err == nil {
		t.Fatal("bad port must fail")
	}
}
