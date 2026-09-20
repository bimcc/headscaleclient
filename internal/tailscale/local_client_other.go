//go:build !darwin

package tailscale

import "tailscale.com/client/local"

func newLocalClient() *local.Client { return &local.Client{} }
