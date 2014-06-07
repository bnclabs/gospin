// Derived from from go-jsonpointer

package failsafe

import (
    "strings"
)

var decoder = strings.NewReplacer("~1", "/", "~0", "~")

func parseJsonPointer(path string) (parts []string) {
    parts = strings.Split(path[1:], "/")
    for i := range parts {
        if strings.Contains(parts[i], "~") {
            parts[i] = decoder.Replace(parts[i])
        }
    }
    return parts
}

func encodeJsonPointer(parts []string) string {
    pathr := make([]rune, 0, 64)

    for _, part := range parts {
        pathr = append(pathr, '/')
        for _, c := range part {
            switch c {
            case '/':
                pathr = append(pathr, '~', '1')
            case '~':
                pathr = append(pathr, '~', '0')
            default:
                pathr = append(pathr, c)
            }
        }
    }
    return string(pathr)
}
