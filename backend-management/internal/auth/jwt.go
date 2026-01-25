package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims represents the JWT claims for authentication tokens.
type Claims struct {
	jwt.RegisteredClaims
	UserID    int64  `json:"userId"`
	Role      string `json:"role"`
	Name      string `json:"name,omitempty"`
	TokenType string `json:"type,omitempty"`
}

// JWTService handles JWT token operations.
type JWTService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// TokenPair represents an access and refresh token pair.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// NewJWTService creates a new JWT service.
func NewJWTService(secret string, accessExpiry, refreshExpiry time.Duration, issuer string) *JWTService {
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil || len(decoded) == 0 {
		decoded = []byte(secret)
	}

	return &JWTService{
		secret:        decoded,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        issuer,
	}
}

// GenerateTokenPair creates a new access and refresh token pair.
func (s *JWTService) GenerateTokenPair(userID int64, email, role, name string) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessToken(userID, email, role, name)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(userID, email)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessExpiry.Seconds()),
	}, nil
}

// GenerateAccessToken creates a signed access token.
func (s *JWTService) GenerateAccessToken(userID int64, email, role, name string) (string, error) {
	now := time.Now()
	nonce, _ := GenerateRandomToken(8)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        nonce,
			Issuer:    s.issuer,
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
		},
		UserID: userID,
		Role:   role,
		Name:   name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken creates a signed refresh token.
func (s *JWTService) GenerateRefreshToken(userID int64, email string) (string, error) {
	now := time.Now()
	nonce, _ := GenerateRandomToken(8)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        nonce,
			Issuer:    s.issuer,
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
		},
		UserID:    userID,
		TokenType: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateAccessToken validates a JWT access token and returns the claims.
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType == "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a JWT refresh token and returns the claims.
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *JWTService) parseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// HashPassword creates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPassword compares a password with its hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateRandomToken generates a random token suitable for password resets.
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
