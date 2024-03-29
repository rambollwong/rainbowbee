package crypto

import (
	"fmt"
	"os"
)

// WeakRsaKeyEnv is an environment variable which, when set, lowers the
// minimum required bits of RSA keys to 512. This should be used exclusively in
// test situations.
const WeakRsaKeyEnv = "RAINBOW_BEE_ALLOW_WEAK_RSA_KEYS"

var MinRsaKeyBits = 2048

var maxRsaKeyBits = 8192

// ErrRsaKeyTooSmall is returned when trying to generate or parse an RSA key
// that's smaller than MinRsaKeyBits bits. In test
var ErrRsaKeyTooSmall error
var ErrRsaKeyTooBig error = fmt.Errorf("rsa keys must be <= %d bits", maxRsaKeyBits)

func init() {
	if _, ok := os.LookupEnv(WeakRsaKeyEnv); ok {
		MinRsaKeyBits = 512
	}

	ErrRsaKeyTooSmall = fmt.Errorf("rsa keys must be >= %d bits to be useful", MinRsaKeyBits)
}
