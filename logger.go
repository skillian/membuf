package membuf

import (
	"github.com/skillian/logging"
)

var logger = logging.GetLogger("membuf")

// bytesZero checks if all of the bytes in the slice are zero values.
func bytesZero(p []byte) bool {
	for _, b := range p {
		if b != 0 {
			return false
		}
	}
	return true
}
