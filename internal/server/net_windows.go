//go:build windows
// +build windows

package server

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func listen(network, address string) (net.Listener, error) {
	switch network {
	case "npipe":
		return winio.ListenPipe(address, &winio.PipeConfig{
			MessageMode:      true,
			InputBufferSize:  65536,
			OutputBufferSize: 65536,
		})
	default:
		return net.Listen(network, address)
	}
}
