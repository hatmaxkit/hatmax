package crypto

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrMissingPrivateKey = errors.New("missing private key")
	ErrMissingPublicKey  = errors.New("missing public key")
)

type TokenClaims struct {
	Subject      string            `json:"sub"`
	SessionID    string            `json:"sid"`
	Audience     string            `json:"aud"`
	Context      map[string]string `json:"ctx,omitempty"`
	ExpiresAt    int64             `json:"exp"`
	AuthzVersion int               `json:"authz_ver,omitempty"`
}

func GenerateToken(claims TokenClaims, privateKey ed25519.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", ErrMissingPrivateKey
	}

	token := paseto.NewToken()

	token.SetAudience(claims.Audience)
	token.SetSubject(claims.Subject)
	token.SetExpiration(time.Unix(claims.ExpiresAt, 0))

	token.SetString("sid", claims.SessionID)

	if claims.Context != nil && len(claims.Context) > 0 {
		ctxBytes, err := json.Marshal(claims.Context)
		if err != nil {
			return "", err
		}
		token.SetString("ctx", string(ctxBytes))
	}

	if claims.AuthzVersion > 0 {
		token.SetString("authz_ver", string(rune(claims.AuthzVersion+'0')))
	}

	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromEd25519(privateKey)
	if err != nil {
		return "", err
	}
	signed := token.V4Sign(secretKey, nil)

	return signed, nil
}

func VerifyToken(tokenString string, publicKey ed25519.PublicKey) (TokenClaims, error) {
	if publicKey == nil {
		return TokenClaims{}, ErrMissingPublicKey
	}

	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())

	pubKey, err := paseto.NewV4AsymmetricPublicKeyFromEd25519(publicKey)
	if err != nil {
		return TokenClaims{}, ErrInvalidToken
	}

	token, err := parser.ParseV4Public(pubKey, tokenString, nil)
	if err != nil {
		if errors.Is(err, paseto.RuleError{}) {
			return TokenClaims{}, ErrTokenExpired
		}
		return TokenClaims{}, ErrInvalidToken
	}

	claims := TokenClaims{}

	audience, err := token.GetAudience()
	if err == nil {
		claims.Audience = audience
	}

	subject, err := token.GetSubject()
	if err == nil {
		claims.Subject = subject
	}

	expiration, err := token.GetExpiration()
	if err == nil {
		claims.ExpiresAt = expiration.Unix()
	}

	sid, err := token.GetString("sid")
	if err == nil {
		claims.SessionID = sid
	}

	ctxStr, err := token.GetString("ctx")
	if err == nil && ctxStr != "" {
		var ctx map[string]string
		if err := json.Unmarshal([]byte(ctxStr), &ctx); err == nil {
			claims.Context = ctx
		}
	}

	authzVerStr, err := token.GetString("authz_ver")
	if err == nil && len(authzVerStr) == 1 {
		claims.AuthzVersion = int(authzVerStr[0] - '0')
	}

	return claims, nil
}

func GenerateSessionID() string {
	return uuid.New().String()
}
