package password

import (
	"crypto/rand"
	"math/big"
)

var alphabet string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-+@"

func Generate(size int) (string, error) {
	buf := make([]byte, size)
	limit := big.NewInt(int64(len(alphabet)))
	for i := 0; i < size; i++ {
		b, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		var d int = int(b.Int64())
		c := alphabet[d]
		buf[i] = byte(c)
	}
	s := string(buf)
	for i := 0; i < size; i++ {
		buf[i] = 0
	}
	return s, nil
}
