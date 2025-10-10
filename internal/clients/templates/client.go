// Package templates provides a path for webpage templates to be lazily loaded from the filesystem.
package templates

import (
	"io/fs"
	"os"
)

type Client struct {
	Config Config
}

func NewClient(c Config) *Client {
	return &Client{
		Config: c,
	}
}

func (c *Client) GetFS() fs.FS {
	path := c.Config.Path
	if path == "" {
		return nil
	}
	return os.DirFS(path)
}
