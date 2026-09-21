package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/resume-platform/backend/internal/models"
)

func testManager() *TokenManager {
	return NewTokenManager("access-secret-for-tests", "refresh-secret-for-tests", 15*time.Minute, 7*24*time.Hour)
}

func TestGenerateAndParseAccessToken_RoundTrip(t *testing.T) {
	tm := testManager()
	token, expiresAt, err := tm.GenerateAccessToken(42, models.RoleCandidate, "jane@example.com")
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token string")
	}
	if time.Until(expiresAt) <= 0 || time.Until(expiresAt) > 16*time.Minute {
		t.Errorf("expiresAt should be ~15 minutes in the future, got %v", expiresAt)
	}

	claims, err := tm.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("unexpected error parsing valid token: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("expected user_id 42, got %d", claims.UserID)
	}
	if claims.Role != models.RoleCandidate {
		t.Errorf("expected role candidate, got %s", claims.Role)
	}
	if claims.Email != "jane@example.com" {
		t.Errorf("expected email jane@example.com, got %s", claims.Email)
	}
}

func TestParseAccessToken_RejectsExpiredToken(t *testing.T) {
	// A TokenManager with a negative access expiry issues already-expired
	// tokens -- this is the cleanest way to deterministically test expiry
	// rejection without sleeping in a test.
	tm := NewTokenManager("access-secret", "refresh-secret", -1*time.Minute, 7*24*time.Hour)
	token, _, err := tm.GenerateAccessToken(1, models.RoleCandidate, "expired@example.com")
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	_, err = tm.ParseAccessToken(token)
	if err == nil {
		t.Fatal("expected an error parsing an expired token, got nil")
	}
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseAccessToken_RejectsTamperedToken(t *testing.T) {
	tm := testManager()
	token, _, err := tm.GenerateAccessToken(1, models.RoleCandidate, "jane@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// flip a character in the signature portion to simulate tampering
	tampered := token[:len(token)-4] + "XXXX"
	if _, err := tm.ParseAccessToken(tampered); err == nil {
		t.Error("expected an error parsing a tampered token, got nil")
	}
}

func TestParseAccessToken_RejectsTokenSignedWithWrongSecret(t *testing.T) {
	tmA := NewTokenManager("secret-a", "refresh-a", 15*time.Minute, time.Hour)
	tmB := NewTokenManager("secret-b", "refresh-b", 15*time.Minute, time.Hour)

	token, _, err := tmA.GenerateAccessToken(1, models.RoleCandidate, "jane@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// a token signed by tmA (secret-a) must never validate against tmB (secret-b)
	if _, err := tmB.ParseAccessToken(token); err == nil {
		t.Error("expected an error validating a token against the wrong signing secret")
	}
}

func TestParseAccessToken_RejectsAlgNoneAttack(t *testing.T) {
	// Classic JWT vulnerability: an attacker crafts a token with alg=none
	// and no signature, hoping a naive verifier accepts it. Our parser
	// must reject this because ParseAccessToken's keyfunc explicitly
	// checks the signing method is HMAC.
	tm := testManager()
	claims := Claims{
		UserID: 1, Role: models.RoleAdmin, Email: "attacker@example.com",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := unsignedToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to construct alg=none token for test: %v", err)
	}

	if _, err := tm.ParseAccessToken(tokenString); err == nil {
		t.Fatal("SECURITY: alg=none token was accepted -- this must always be rejected")
	}
}

func TestGenerateRefreshToken_ProducesHighEntropyUniqueTokens(t *testing.T) {
	tm := testManager()
	raw1, hash1, _, err := tm.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw2, hash2, _, err := tm.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if raw1 == raw2 {
		t.Error("two refresh tokens should never be identical (would indicate a broken RNG)")
	}
	if len(raw1) < 32 {
		t.Errorf("refresh token should be high-entropy (32+ hex chars), got length %d", len(raw1))
	}
	if hash1 == raw1 {
		t.Error("the stored hash must not equal the raw token")
	}
	if hash1 == hash2 {
		t.Error("hashes of two different raw tokens must differ")
	}
}

func TestHashToken_DeterministicAndOneWay(t *testing.T) {
	h1 := HashToken("some-raw-refresh-token")
	h2 := HashToken("some-raw-refresh-token")
	if h1 != h2 {
		t.Error("HashToken must be deterministic: same input always produces the same hash (needed for DB lookup by hash)")
	}
	if h1 == "some-raw-refresh-token" {
		t.Error("HashToken output must not equal its input")
	}

	h3 := HashToken("a-different-token")
	if h1 == h3 {
		t.Error("different inputs must produce different hashes")
	}
}
