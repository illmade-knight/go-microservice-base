package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/illmade-knight/go-microservice-base/pkg/response"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// contextKey is a private type to prevent collisions with other context keys.
type contextKey string

// userContextKey is the key used to store the authenticated user's ID from the JWT.
const userContextKey contextKey = "userID"

// NewJWKSAuthMiddleware is the modern, secure constructor for creating JWT authentication middleware.
// It validates asymmetric RS256 tokens by fetching public keys from a JWKS endpoint.
// This should be the default choice for all new services.
func NewJWKSAuthMiddleware(jwksURL string) (func(http.Handler) http.Handler, error) {
	// Create a new JWK cache that will automatically fetch and refresh the keys.
	// This is done once on startup for efficiency.
	cache := jwk.NewCache(context.Background())
	err := cache.Register(jwksURL, jwk.WithRefreshInterval(15*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

	// Pre-fetch the keys on startup to ensure the identity service is reachable.
	// This makes the service fail-fast if the JWKS endpoint is misconfigured.
	_, err = cache.Refresh(context.Background(), jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to perform initial JWKS fetch: %w", err)
	}

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

			// The keyfunc is called by the JWT library during parsing.
			// It fetches the key set from our cache and finds the key that
			// matches the token's `kid` (Key ID) header.
			keyFunc := func(token *jwt.Token) (interface{}, error) {
				keySet, err := cache.Get(r.Context(), jwksURL)
				if err != nil {
					return nil, fmt.Errorf("failed to get key set from cache: %w", err)
				}

				keyID, ok := token.Header["kid"].(string)
				if !ok {
					return nil, fmt.Errorf("token missing 'kid' header")
				}

				key, found := keySet.LookupKeyID(keyID)
				if !found {
					return nil, fmt.Errorf("key with ID '%s' not found in JWKS", keyID)
				}

				var rawKey interface{}
				if err := key.Raw(&rawKey); err != nil {
					return nil, fmt.Errorf("failed to get raw public key: %w", err)
				}
				return rawKey, nil
			}

			// Parse the token, providing our keyfunc to find the correct public key.
			// We now explicitly require the RS256 signing method.
			token, err := jwt.Parse(tokenString, keyFunc, jwt.WithValidMethods([]string{"RS256"}))

			if err != nil {
				response.WriteJSONError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: Invalid token (%s)", err.Error()))
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
	}, nil
}

// DEPRECATED: NewLegacySharedSecretAuthMiddleware uses a symmetric HS256 shared secret for JWT validation.
// This pattern is less secure as it requires sharing the secret with all services.
// It is retained for backward compatibility only and should NOT be used for new services.
// Use NewJWKSAuthMiddleware instead.
func NewLegacySharedSecretAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
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
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userContextKey).(string)
	return userID, ok
}

// ContextWithUserID is a helper function for tests to inject a user ID
// into a context, simulating a successful authentication.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userContextKey, userID)
}
