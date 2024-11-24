//go:build windows

package plugin

import "github.com/google/go-tpm/tpm2/transport"

func openTPM(_ ...string) (transport.TPMCloser, error) {
	return transport.OpenTPM()
}
