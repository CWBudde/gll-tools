package mesh

import (
	"fmt"
	"io"
	"strings"
)

func WriteMTL(w io.Writer, name string) error {
	if name == "" {
		name = defaultBalloonName
	}
	name = sanitizeOBJName(name)
	fmt.Fprintf(w, "newmtl %s\n", name)
	fmt.Fprint(w, "Ka 0.2 0.2 0.2\n")
	fmt.Fprint(w, "Kd 0.8 0.8 0.8\n")
	fmt.Fprint(w, "Ks 0.05 0.05 0.05\n")
	fmt.Fprint(w, "Ns 10\n")
	fmt.Fprint(w, "illum 2\n")
	return nil
}

func SanitizeMTLFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "balloon.mtl"
	}
	return sanitizeOBJName(name) + ".mtl"
}
