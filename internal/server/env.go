// Helper konfigurasi dari environment.
package server

import "strings"

// SplitList memecah "a, b ,c" dari env menjadi irisan bersih.
func SplitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, strings.TrimRight(p, "/"))
		}
	}
	return out
}
