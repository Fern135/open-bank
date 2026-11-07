package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/hotp"
)

const totpPeriod = 30 // seconds, what authenticator apps use

// EncryptSecret seals a two-factor secret with AES-256-GCM for storage.
func EncryptSecret(key []byte, secret string) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	return gcm.Seal(nonce, nonce, []byte(secret), nil), nil
}

// DecryptSecret is the reverse of EncryptSecret.
func DecryptSecret(key, sealed []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("sealed secret is too short")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// VerifyTOTP checks a code against the current time step, allowing one step
// either side for clock drift. It returns the matched counter, and rejects
// anything at or below lastCounter so a code can't be used twice.
func VerifyTOTP(secret, code string, lastCounter int64, now time.Time) (int64, bool) {
	current := now.Unix() / totpPeriod
	for counter := current - 1; counter <= current+1; counter++ {
		if counter <= lastCounter {
			continue
		}
		want, err := hotp.GenerateCodeCustom(secret, uint64(counter), hotp.ValidateOpts{
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err == nil && subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return counter, true
		}
	}
	return 0, false
}
