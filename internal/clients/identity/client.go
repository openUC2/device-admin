// Package identity loads and exposes identity information about the machine
package identity

import (
	"cmp"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
)

type Config struct {
	MachineNamePath string
}

type Client struct {
	Config Config

	l godest.Logger
}

func NewClient(c Config, l godest.Logger) *Client {
	return &Client{
		Config: c,
		l:      l,
	}
}

func (c *Client) GetMachineName() (name string, err error) {
	p := cmp.Or(c.Config.MachineNamePath, "/run/machine-name")
	lines, err := readFile(p)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't read machine name file %s", p)
	}
	return strings.Join(lines, ""), nil
}

func readFile(filePath string) (lines []string, err error) {
	if lines, err = readLines(filePath); err != nil {
		return nil, errors.Wrapf(err, "couldn't read file %s", filePath)
	}
	for i, l := range lines {
		before, _, _ := strings.Cut(l, "#")
		lines[i] = strings.TrimSpace(before)
	}
	return lines, nil
}

func readLines(filePath string) ([]string, error) {
	contents, err := os.ReadFile(filePath) //nolint:gosec // We trust this file
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read file %s", filePath)
	}

	return strings.Split(string(contents), "\n"), nil
}

func (c *Client) GetHostname() (name string, err error) {
	return os.Hostname()
}
