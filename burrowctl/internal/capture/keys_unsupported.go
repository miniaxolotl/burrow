//go:build !linux

package capture

import "errors"

func (OSKeys) Key(string) ([]byte, error) {
	return nil, errors.New("capture is unavailable on this platform until a non-interactive secure key adapter is verified; ordinary tunneling remains available")
}
