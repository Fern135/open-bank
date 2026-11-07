package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/hotp"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestAccessTokenRoundTrip(t *testing.T) {
	token, err := IssueAccessToken(secret, "user", "session")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseAccessToken(secret, token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user" || claims.SessionID != "session" {
		t.Errorf("got %+v", claims)
	}
}

func TestParseRejectsBadTokens(t *testing.T) {
	good, _ := IssueAccessToken(secret, "user", "session")
	wrongKey, _ := IssueAccessToken([]byte("another-secret-another-secret-00"), "user", "session")
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{SessionID: "session"}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}).SignedString(secret)
	noExpiry, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer},
	}).SignedString(secret)

	tests := map[string]string{
		"wrong key": wrongKey,
		"alg none":  none,
		"expired":   expired,
		"no expiry": noExpiry,
		"tampered":  good[:len(good)-2] + "xx",
		"not a jwt": "hello",
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseAccessToken(secret, token); err == nil {
				t.Error("token was accepted")
			}
		})
	}
}

func TestRefreshTokenFormat(t *testing.T) {
	plain, hash := NewRefreshSecret()
	id, got, err := SplitRefreshToken(FormatRefreshToken("session", plain))
	if err != nil || id != "session" || got != plain {
		t.Fatalf("got %q %q %v", id, got, err)
	}
	if !HashesEqual(HashRefreshSecret(got), hash) {
		t.Error("hash does not match its secret")
	}
	for _, bad := range []string{"", "nodot", ".secret", "session."} {
		if _, _, err := SplitRefreshToken(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestSecretEncryption(t *testing.T) {
	key := secret
	sealed, err := EncryptSecret(key, "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sealed), "JBSWY3DPEHPK3PXP") {
		t.Error("secret is stored in the clear")
	}
	if got, err := DecryptSecret(key, sealed); err != nil || got != "JBSWY3DPEHPK3PXP" {
		t.Errorf("got %q %v", got, err)
	}

	sealed[len(sealed)-1] ^= 1
	if _, err := DecryptSecret(key, sealed); err == nil {
		t.Error("tampered secret was accepted")
	}
}

func TestVerifyTOTP(t *testing.T) {
	const totpSecret = "JBSWY3DPEHPK3PXP"
	now := time.Unix(1_800_000_000, 0)
	counter := now.Unix() / totpPeriod
	code := func(c int64) string {
		value, err := hotp.GenerateCodeCustom(totpSecret, uint64(c), hotp.ValidateOpts{
			Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil {
			t.Fatal(err)
		}
		return value
	}

	if got, ok := VerifyTOTP(totpSecret, code(counter), 0, now); !ok || got != counter {
		t.Errorf("current code: got %d %v", got, ok)
	}
	if _, ok := VerifyTOTP(totpSecret, code(counter-1), 0, now); !ok {
		t.Error("code from the previous step should pass for clock drift")
	}
	if _, ok := VerifyTOTP(totpSecret, code(counter-2), 0, now); ok {
		t.Error("code from two steps ago was accepted")
	}
	if _, ok := VerifyTOTP(totpSecret, code(counter), counter, now); ok {
		t.Error("a code was accepted twice")
	}
	if _, ok := VerifyTOTP(totpSecret, "000000x", 0, now); ok {
		t.Error("a malformed code was accepted")
	}
}
