package utils

import "golang.org/x/crypto/bcrypt"

// bcrypt cost 12 is a deliberate choice: higher than the library default of
// 10, giving meaningfully more brute-force resistance while still hashing
// in well under 200ms on typical server hardware.
const bcryptCost = 12

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
