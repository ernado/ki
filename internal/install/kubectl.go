package install

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

type KubectlApplyOptions struct {
	File       string
	Kubeconfig string
}

func KubectlApply(opt KubectlApplyOptions) error {
	fmt.Println("> kubectl apply -f", opt.File)
	cmd := exec.Command("kubectl", "apply", "-f", opt.File)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if opt.Kubeconfig != "" {
		cmd.Env = appendEnv(os.Environ(), "KUBECONFIG", opt.Kubeconfig)
	}
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl apply")
	}
	return nil
}
