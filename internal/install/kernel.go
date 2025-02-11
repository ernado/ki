package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/go-faster/errors"
)

func LoadKernelModules(name string, modules ...string) error {
	for _, module := range modules {
		fmt.Println("> Loading module", module)
		cmd := exec.Command("modprobe", module)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.Wrapf(err, "modprobe %s", module)
		}
	}
	// Persist in named configuration.
	fileName := filepath.Join("/etc/modules-load.d", name+".conf")
	fmt.Printf("> Writing %s\n", fileName)
	var out []byte
	for _, module := range modules {
		out = append(out, module...)
		out = append(out, '\n')
	}
	if err := os.WriteFile(fileName, out, 0600); err != nil {
		return errors.Wrap(err, "write")
	}
	return nil
}

func ConfigureKernelParameters(name string, params map[string]any) error {
	fmt.Printf("> Configuring kernel parameters for %s\n", name)
	fileName := filepath.Join("/etc/sysctl.d", name+".conf")
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Printf("> Writing %s\n", fileName)
	var out []byte
	for _, key := range keys {
		out = append(out, key...)
		out = append(out, '=')
		out = append(out, fmt.Sprintf("%v", params[key])...)
		out = append(out, '\n')
	}
	if err := os.WriteFile(fileName, out, 0600); err != nil {
		return errors.Wrap(err, "write")
	}
	// Reload.
	cmd := exec.Command("sysctl", "--system")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "sysctl --system")
	}
	return nil
}
