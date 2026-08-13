package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// TTL de los tokens de un solo uso. El de reset de password es más corto
// a propósito: una vez pedido, la ventana para usarlo debería ser chica.
const (
	EmailVerificationTokenTTL = 24 * time.Hour
	PasswordResetTokenTTL     = 1 * time.Hour
)

// GenerateToken crea un token aleatorio de 32 bytes (256 bits de entropía)
// para links de un solo uso. Devuelve el valor crudo (el único que viaja
// en el link del email) y su hash SHA-256 (lo único que se guarda en la
// base). No usamos bcrypt acá: bcrypt está pensado para contraseñas de
// baja entropía elegidas por humanos; un token aleatorio de 256 bits ya
// es imposible de fuerza-bruta, y necesitamos poder buscarlo por igualdad
// exacta en la base, algo que bcrypt no permite indexar.
func GenerateToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("auth: generando token: %w", err)
	}
	raw = hex.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

// HashToken calcula el hash SHA-256 de un token en texto plano.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
