package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// VerifyTOTP implements RFC 6238 with a one-step clock allowance. The caller
// must persist the accepted step atomically to reject replay.
func VerifyTOTP(secret, code string, now time.Time, lastStep int64) (int64, error) {
	if len(code) != 6 {
		return 0, errors.New("invalid one-time code")
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid one-time code")
		}
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil || len(key) < 20 {
		return 0, errors.New("invalid authenticator secret")
	}
	step := now.Unix() / 30
	for _, candidate := range []int64{step, step - 1, step + 1} {
		if candidate <= lastStep {
			continue
		}
		var counter [8]byte
		binary.BigEndian.PutUint64(counter[:], uint64(candidate))
		mac := hmac.New(sha1.New, key)
		_, _ = mac.Write(counter[:])
		sum := mac.Sum(nil)
		offset := sum[len(sum)-1] & 15
		value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
		expected := fmt.Sprintf("%06d", value%1000000)
		if hmac.Equal([]byte(expected), []byte(code)) {
			return candidate, nil
		}
	}
	return 0, errors.New("invalid or used one-time code")
}
