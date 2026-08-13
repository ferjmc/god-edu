package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionDuration es cuánto dura una sesión antes de pedir login de nuevo.
// No hay refresh token: para este tamaño de plataforma, re-loguearse cada
// semana es una fricción aceptable a cambio de no mantener otro mecanismo.
const SessionDuration = 7 * 24 * time.Hour

var ErrInvalidToken = errors.New("auth: token inválido o expirado")

// Claims son los datos propios que viajan dentro del JWT de sesión.
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTManager emite y valida los JWT de sesión, firmados con HS256.
type JWTManager struct {
	secret []byte
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: []byte(secret)}
}

// Issue genera un JWT de sesión para el usuario dado.
func (m *JWTManager) Issue(userID int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(SessionDuration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse valida la firma y expiración de un JWT y devuelve sus claims.
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: método de firma inesperado: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
