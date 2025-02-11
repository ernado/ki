package install

import (
	"encoding/json"
	"os/exec"

	"github.com/go-faster/errors"
)

type Route struct {
	Dst      string   `json:"dst"`
	Gateway  string   `json:"gateway,omitempty"`
	Dev      string   `json:"dev"`
	Protocol string   `json:"protocol"`
	PrefSrc  string   `json:"prefsrc"`
	Metric   int      `json:"metric"`
	Flags    []any    `json:"flags"`
	Metrics  []Metric `json:"metrics,omitempty"`
	Scope    string   `json:"scope,omitempty"`
}

type Metric struct {
	MTU int `json:"mtu"`
}

func GetDefaultGatewayIP() (string, error) {
	// This is valid for hetzher.
	cmd := exec.Command("ip", "-j", "route", "show", "default")
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "ip route show default")
	}
	var routes []Route
	if err := json.Unmarshal(out, &routes); err != nil {
		return "", errors.Wrap(err, "unmarshal")
	}
	for _, route := range routes {
		if route.Dst == "default" {
			return route.PrefSrc, nil
		}
	}
	return "", errors.New("default gateway not found")
}
