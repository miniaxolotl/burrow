package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	Plan   string `json:"plan"`
}

func Register(ctx context.Context, pg *PostgresClient, jwtSecret, email, password string) (*User, string, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", fmt.Errorf("hash password: %w", err)
	}
	user, err := pg.CreateUser(ctx, email, string(hash))
	if err != nil {
		return nil, "", "", fmt.Errorf("create user: %w", err)
	}
	accessToken, err := generateAccessToken(user, jwtSecret)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	_, err = pg.CreateSession(ctx, user.ID, refreshToken, "", "", time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, "", "", fmt.Errorf("create session: %w", err)
	}
	return user, accessToken, refreshToken, nil
}

func Login(ctx context.Context, pg *PostgresClient, jwtSecret, email, password string) (*User, string, string, error) {
	user, err := pg.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", fmt.Errorf("invalid credentials")
	}
	accessToken, err := generateAccessToken(user, jwtSecret)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	_, err = pg.CreateSession(ctx, user.ID, refreshToken, "", "", time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, "", "", fmt.Errorf("create session: %w", err)
	}
	return user, accessToken, refreshToken, nil
}

func RefreshToken(ctx context.Context, pg *PostgresClient, jwtSecret, refreshToken string) (string, error) {
	session, err := pg.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token")
	}
	user, err := pg.GetUserByID(ctx, session.UserID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	accessToken, err := generateAccessToken(user, jwtSecret)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}
	return accessToken, nil
}

func Logout(ctx context.Context, pg *PostgresClient, refreshToken string) error {
	return pg.DeleteSession(ctx, refreshToken)
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func ValidateAccessToken(tokenString, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func generateAccessToken(user *User, jwtSecret string) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: user.ID,
		Role:   user.Role,
		Email:  user.Email,
		Plan:   user.Plan,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}