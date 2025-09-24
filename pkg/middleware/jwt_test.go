package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/illmade-knight/go-microservice-base/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "my-test-secret"

// createTestToken generates a JWT for testing purposes.
func createTestToken(userID string, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	return token.SignedString([]byte(secret))
}

func TestJWTAuthMiddleware(t *testing.T) {
	// The handler that will be protected by the middleware.
	// It checks if the userID was correctly placed in the context.
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		require.True(t, ok, "userID should be in the context")
		require.Equal(t, "user-123", userID)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	jwtMiddleware := middleware.NewJWTAuthMiddleware(testSecret)
	protectedHandler := jwtMiddleware(testHandler)

	t.Run("Success - Valid Token", func(t *testing.T) {
		token, err := createTestToken("user-123", testSecret)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "OK", rr.Body.String())
	})

	t.Run("Failure - No Auth Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.JSONEq(t, `{"error":"Unauthorized: Missing Authorization header"}`, rr.Body.String())
	})

	t.Run("Failure - Invalid Token Format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "invalid-format")
		rr := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.JSONEq(t, `{"error":"Unauthorized: Invalid token format"}`, rr.Body.String())
	})

	t.Run("Failure - Invalid Signature", func(t *testing.T) {
		token, err := createTestToken("user-123", "a-different-secret")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.JSONEq(t, `{"error":"Unauthorized: Invalid token"}`, rr.Body.String())
	})
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	userID := "test-user"

	// Test adding a user ID to the context
	ctxWithUser := middleware.ContextWithUserID(ctx, userID)
	retrievedID, ok := middleware.GetUserIDFromContext(ctxWithUser)

	assert.True(t, ok)
	assert.Equal(t, userID, retrievedID)

	// Test retrieving from a context without the user ID
	_, ok = middleware.GetUserIDFromContext(ctx)
	assert.False(t, ok)
}
