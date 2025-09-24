package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/illmade-knight/go-microservice-base/pkg/response"
)

// contextKey is a private type to prevent collisions with other context keys.
type contextKey string

// userContextKey is the key used to store the authenticated user's ID from the JWT.
const userContextKey contextKey = "userID"

// NewJWTAuthMiddleware is a constructor that creates a JWT authentication middleware.
// It takes the JWT secret key as an argument and returns a standard middleware function.
func NewJWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized: Missing Authorization header")
				return
			}

			tokenString, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				response.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized: Invalid token format")
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				response.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized: Invalid token")
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				userID, ok := claims["sub"].(string)
				if !ok || userID == "" {
					response.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized: Invalid user ID in token")
					return
				}

				ctx := context.WithValue(r.Context(), userContextKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				response.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized: Invalid token claims")
			}
		})
	}
}

// GetUserIDFromContext safely retrieves the user ID from the request context.
// This function is intended for use by downstream handlers.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userContextKey).(string)
	return userID, ok
}

// ContextWithUserID is a helper function for tests to inject a user ID
// into a context, simulating a successful authentication.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userContextKey, userID)
}
