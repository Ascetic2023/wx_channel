package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const InsecureDefaultJWTSecret = "your-secret-key-change-this-in-prod"

var JWTSecret = []byte(InsecureDefaultJWTSecret)

// InitJWTSecretFromEnv loads JWT secret from environment and validates it.
func InitJWTSecretFromEnv() error {
	secret := strings.TrimSpace(os.Getenv("HUB_JWT_SECRET"))
	if secret == "" {
		return fmt.Errorf("HUB_JWT_SECRET is required")
	}
	if len(secret) < 32 {
		return fmt.Errorf("HUB_JWT_SECRET must be at least 32 characters")
	}
	if secret == InsecureDefaultJWTSecret {
		return fmt.Errorf("HUB_JWT_SECRET must not use insecure default value")
	}
	JWTSecret = []byte(secret)
	return nil
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token for a user
func GenerateToken(userID uint, email, role string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// AuthRequired is a middleware that validates the JWT token
func AuthRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenStr := bearerToken[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Set user info in context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_role", claims.Role)

		next(w, r.WithContext(ctx))
	}
}

// OptionalAuth allows access but sets context if token is present
func OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			bearerToken := strings.Split(authHeader, " ")
			if len(bearerToken) == 2 && bearerToken[0] == "Bearer" {
				tokenStr := bearerToken[1]
				claims := &Claims{}
				token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
					return JWTSecret, nil
				})
				if err == nil && token.Valid {
					ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
					ctx = context.WithValue(ctx, "user_email", claims.Email)
					ctx = context.WithValue(ctx, "user_role", claims.Role)
					r = r.WithContext(ctx)
				}
			}
		}
		next(w, r)
	}
}

// AdminRequired requires both authentication and 'admin' role
func AdminRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// First run AuthRequired check manually or assume it's chained?
		// Better: Chain it. `AdminRequired(AuthRequired(handler))`?
		// No, usually middlewares are chained.
		// If used as `AdminRequired(handler)`, we must ensure `w` and `r` have user info or parse token again.
		// To avoid parsing twice, let's assume `AuthRequired` wraps this, OR this wraps `AuthRequired`, OR we duplicate logic.
		// Best practice: `http.HandleFunc(..., AuthRequired(AdminRequired(handler)))`

		role := r.Context().Value("user_role")
		if role == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if role.(string) != "admin" {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
