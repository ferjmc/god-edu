package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	// bcrypt ignora silenciosamente los bytes que pasan de 72; en vez de
	// confiar en eso, lo rechazamos antes para que el usuario sepa por qué.
	maxPasswordLength = 72
)

var (
	ErrPasswordTooShort = errors.New("auth: la contraseña debe tener al menos 8 caracteres")
	ErrPasswordTooLong  = errors.New("auth: la contraseña no puede superar los 72 caracteres")
)

// HashPassword valida la longitud y devuelve el hash bcrypt de la contraseña.
func HashPassword(password string) (string, error) {
	if len(password) < minPasswordLength {
		return "", ErrPasswordTooShort
	}
	if len(password) > maxPasswordLength {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compara una contraseña en texto plano contra un hash bcrypt.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
