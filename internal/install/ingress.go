package install

import (
	_ "embed"
	"os"
	"os/exec"
	"strings"
)

//go:embed ingress.yaml
var ingressDefinition string

func DefaultIngress() error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(ingressDefinition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
