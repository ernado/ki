// Package install wraps install steps.
package install

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-faster/errors"
)

//go:embed ki.service
var kiService string

type ServiceOptions struct {
	Join bool
}

func Service(opt ServiceOptions) error {
	// Install as oneshot systemd service.
	// Write to /etc/systemd/system/ki.service.
	// Reload systemd.
	// Enable and start ki.service.
	// Service should be run in background.
	fmt.Println("> Install ki service")
	{
		// Create /etc/ki.conf with OPTIONS.
		var b strings.Builder
		b.WriteString("OPTIONS=")
		if opt.Join {
			b.WriteString("--join")
		}
		b.WriteString("\n")
		if err := os.WriteFile("/etc/ki.conf", []byte(b.String()), 0600); err != nil {
			return errors.Wrap(err, "write ki.conf")
		}
	}
	if err := os.WriteFile("/etc/systemd/system/ki.service", []byte(kiService), 0600); err != nil {
		return errors.Wrap(err, "write ki.service")
	}
	{
		cmd := exec.Command("systemctl", "daemon-reload")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "daemon-reload")
		}
	}
	{
		cmd := exec.Command("systemctl", "start", "--no-block", "ki.service")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "start ki.service")
		}
	}
	fmt.Println("> ki service installed")

	return nil
}
