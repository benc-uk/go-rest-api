// --------------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2022
// Licensed under the MIT License.
//
// JWTValidator middleware & wrapper for securing routes with OAuth2 and JWT auth
// --------------------------------------------------------------------------------

package auth

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWTValidator is a struct that can be used to protect routes
type JWTValidator struct {
	clientID string
	jwksURL  string
	scope    string
}

type PassthroughValidator struct {
}

type Validator interface {
	Middleware(next http.Handler) http.Handler
	Protect(next http.HandlerFunc) http.HandlerFunc
}

// NewJWTValidator creates a new JWTValidator struct
func NewJWTValidator(clientID string, jwksURL string, scope string) JWTValidator {
	return JWTValidator{
		clientID: clientID,
		jwksURL:  jwksURL,
		scope:    scope,
	}
}

func NewPassthroughValidator() PassthroughValidator {
	return PassthroughValidator{}
}

// Middleware returns middleware to enforce JWT auth on all routes
func (v JWTValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validateRequest(r, v.clientID, v.jwksURL, v.scope) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Protect can be added around any route handler to enforce JWT auth
func (v JWTValidator) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validateRequest(r, v.clientID, v.jwksURL, v.scope) {
			w.WriteHeader(http.StatusUnauthorized)
			 return
		}

		next.ServeHTTP(w, r)
	}
}

// PassthroughValidator middleware does nothing :)
func (v PassthroughValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// PassthroughValidator protect function does nothing :)
func (v PassthroughValidator) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}
}

// validateRequest is an internal function to validate a request
func validateRequest(r *http.Request, clientID string, jwksURL string, scope string) bool {
	// Get auth header & bearer scheme
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) == 0 {
		return false
	}

	// Split header into scheme & B64 token	string
	authParts := strings.Split(authHeader, " ")
	if len(authParts) != 2 {
		return false
	}

	if strings.ToLower(authParts[0]) != "bearer" {
		return false
	}

	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
		RefreshInterval: time.Duration(1) * time.Hour,
	})
	if err != nil {
		log.Printf("### Auth: Failed to get the JWKS. Error: %s", err)
		return false
	}

	// Parse the JWT string using the key fetched from the JWKS
	token, err := jwt.Parse(authParts[1], jwks.Keyfunc)
	if err != nil {
		log.Printf("### Auth: Failed to parse the JWT. Error: %s", err)
		return false
	}

	claims := token.Claims.(jwt.MapClaims)

	// Check the scope includes the app scope
	if !strings.Contains(claims["scp"].(string), scope) {
		log.Printf("### Auth: Scope '%s' is missing from token scope '%s'", scope, claims["scp"])
		return false
	}

	// Check the token audience is the client id, this might have already been done by jwt.Parse
	if claims["aud"] != clientID {
		log.Printf("### Auth: Token audience '%s' does not match '%s'", claims["aud"], clientID)
		return false
	}

	return true
}
