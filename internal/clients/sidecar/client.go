// Package sidecar provides an interface for the machine-admin sidecar's varlink service
package sidecar

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"
)

type Config struct {
	Address string
}

type Client struct {
	Address string
}

func NewClient(conf Config) *Client {
	return &Client{Address: conf.Address}
}

func (c *Client) Open(ctx context.Context) (conn *varlink.Connection, err error) {
	if conn, err = varlink.NewConnection(ctx, c.Address); err != nil {
		return conn, errors.Wrap(err, "couldn't connect to the sidecar over varlink")
	}
	return conn, nil
}

func CloseConn(conn *varlink.Connection, l godest.Logger) {
	if conn == nil {
		return
	}
	if err := conn.Close(); err != nil {
		l.Error(errors.New("couldn't close connection to sidecar"))
	}
}
