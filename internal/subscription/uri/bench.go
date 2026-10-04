package uri

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// benchURIs builds n syntactically valid URIs across all supported schemes.
func BenchURIs(n int) []string {
	const uuid = "11111111-2222-4333-8444-555555555555"
	schemes := []func(i int) string{
		func(i int) string {
			return fmt.Sprintf("vless://%s@h%d.example:443?security=tls&sni=s.example&type=ws&path=/w#n%d", uuid, i, i)
		},
		func(i int) string {
			v, _ := json.Marshal(map[string]any{
				"v": "2", "ps": fmt.Sprintf("n%d", i), "add": fmt.Sprintf("m%d.example", i),
				"port": "443", "id": uuid, "aid": "0", "scy": "auto", "net": "ws",
				"tls": "tls", "host": "cdn.example", "path": "/w", "sni": "cdn.example",
			})
			return "vmess://" + base64.RawURLEncoding.EncodeToString(v)
		},
		func(i int) string {
			return fmt.Sprintf("trojan://pw@h%d.example:443?sni=s.example#n%d", i, i)
		},
		func(i int) string {
			return fmt.Sprintf("ss://YWVzLTI1Ni1nY206cGFzcw==@h%d.example:8388#n%d", i, i)
		},
		func(i int) string {
			return fmt.Sprintf("hy2://%s@h%d.example:443?sni=s.example#n%d", uuid, i, i)
		},
		func(i int) string {
			return fmt.Sprintf("tuic://%s:pw@h%d.example:8443?congestion_control=bbr#n%d", uuid, i, i)
		},
		func(i int) string {
			return fmt.Sprintf("anytls://pw@h%d.example:443#n%d", i, i)
		},
		func(i int) string {
			return fmt.Sprintf("ssh://root@h%d.example:22?password=pw#n%d", i, i)
		},
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, schemes[i%len(schemes)](i))
	}
	return out
}
