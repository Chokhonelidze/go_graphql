package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"okta_config"
	"azure_functions"
)

var logger *zap.Logger

// JWTPayload holds the decoded JWT payload
type JWTPayload struct {
	Payload map[string]interface{}
}

// JWTBearer handles JWT authentication
type JWTBearer struct {
	autoError bool
	payload   map[string]interface{}
	jwt       *JWTPayload
}

// ValidateRemotely validates token with Okta introspection endpoint
func ValidateRemotely(token, clientID, clientSecret string) (map[string]interface{}, error) {
	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("token", token)

	introspectURL := okta_config.OktaSettings.IntrospectURI

	req, err := http.NewRequest("POST", introspectURL, strings.NewReader(data.Encode()))
	if err != nil {
		logger.Error("Failed to create request", zap.Error(err))
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to make request", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", zap.Error(err))
		return nil, err
	}

	logger.Info("Response from introspection endpoint", zap.Int("status_code", resp.StatusCode))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("introspection returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		logger.Error("Failed to unmarshal response", zap.Error(err))
		return nil, err
	}

	return result, nil
}

// CheckAccessToken validates JWT token locally
func CheckAccessToken(token string) (map[string]interface{}, error) {
	secretKey, err := azure_functions.GetSecretFromVault(os.Getenv("CLIENT_SECRET_AP"))
	if err != nil {
		logger.Error("Failed to get secret from vault", zap.Error(err))
		return nil, err
	}

	algorithm := os.Getenv("ALGORITHM")
	if algorithm == "" {
		algorithm = "HS256"
	}

	claims := jwt.MapClaims{}
	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			logger.Error("Invalid token signature", zap.Error(err))
			return nil, fmt.Errorf("invalid token")
		}
		logger.Error("Failed to parse token", zap.Error(err))
		return nil, err
	}

	if !tkn.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return map[string]interface{}(claims), nil
}

// DecodeJWT decodes and validates JWT token remotely
func DecodeJWT(token string) (map[string]interface{}, error) {
	logger.Info("Decoding JWT")

	clientID := os.Getenv("OKTA_CLIENT_ID_AP")
	clientSecret := os.Getenv("OKTA_CLIENT_SECRET_AP")

	decodedToken, err := ValidateRemotely(token, clientID, clientSecret)
	if err != nil {
		logger.Error("Error decoding JWT", zap.Error(err))
		return map[string]interface{}{}, nil
	}

	// Check expiration
	if exp, ok := decodedToken["exp"].(float64); ok {
		if exp >= float64(time.Now().Unix()) {
			return decodedToken, nil
		}
	}

	return map[string]interface{}{}, nil
}

// NewJWTBearer creates a new JWTBearer instance
func NewJWTBearer(autoError bool) *JWTBearer {
	return &JWTBearer{
		autoError: autoError,
		payload:   make(map[string]interface{}),
		jwt:       &JWTPayload{Payload: make(map[string]interface{})},
	}
}

// GetPayload returns the current payload
func (jb *JWTBearer) GetPayload() map[string]interface{} {
	return jb.payload
}

// SetPayload sets the payload
func (jb *JWTBearer) SetPayload(value map[string]interface{}) {
	jb.payload = value
}

// VerifyJWT verifies the JWT token
func (jb *JWTBearer) VerifyJWT(jwtoken string) bool {
	isTokenValid := false

	payload, err := DecodeJWT(jwtoken)
	if err != nil {
		logger.Error("Error verifying JWT", zap.Error(err))
		return false
	}

	if payload != nil && len(payload) > 0 {
		isTokenValid = true
		jb.jwt.Payload = payload
	}

	return isTokenValid
}

