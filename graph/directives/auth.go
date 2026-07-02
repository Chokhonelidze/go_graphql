package directives

import (
	"app/core"
	"app/graph/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func getRank(role string) int {
	switch role {
	case "User":
		return 1
	case "Power User":
		return 2
	case "Admin":
		return 3
	default:
		return 0
	}
}

func getClaimString(claims jwt.MapClaims, keys ...string) string {
	for _, key := range keys {
		value, exists := claims[key]
		if !exists || value == nil {
			continue
		}

		switch v := value.(type) {
		case string:
			if v != "" {
				return v
			}
		case fmt.Stringer:
			str := v.String()
			if str != "" {
				return str
			}
		}
	}

	return ""
}

func getClaimIntSlice(claims jwt.MapClaims, keys ...string) []int {
	for _, key := range keys {
		value, exists := claims[key]
		if !exists || value == nil {
			continue
		}

		switch v := value.(type) {
		case []int:
			return v
		case []int32:
			result := make([]int, 0, len(v))
			for _, n := range v {
				result = append(result, int(n))
			}
			return result
		case []interface{}:
			result := make([]int, 0, len(v))
			for _, item := range v {
				switch n := item.(type) {
				case float64:
					result = append(result, int(n))
				case int:
					result = append(result, n)
				case int32:
					result = append(result, int(n))
				case int64:
					result = append(result, int(n))
				case string:
					parsed, err := strconv.Atoi(n)
					if err == nil {
						result = append(result, parsed)
					}
				}
			}
			return result
		}
	}

	return nil
}

var tokenCache = make(map[string]map[string]interface{})

func ValidateRemotely(token, clientID, clientSecret string) (map[string]interface{}, error) {
	// Check cache first
	fmt.Printf("Validating token: %s\n", token)
	if cached, exists := tokenCache[token]; exists {
		if cached, exists := tokenCache[token]; exists {
			if exp, ok := cached["exp"].(float64); ok {
				if int64(exp) > time.Now().Unix() {
					fmt.Printf("reading token from cache\n")
					return cached, nil
				}
			}
			// Token expired, remove from cache and continue to validate
			delete(tokenCache, token)
		}
		return cached, nil
	}

	fmt.Printf("Token not in cache, validating remotely\n")

	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("token", token)

	introspectURL := core.OktaSettings.IntrospectURI

	req, err := http.NewRequest("POST", introspectURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("introspection returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	// Store in cache
	tokenCache[token] = result
	return result, nil
}
func Auth(ctx context.Context, obj interface{}, next graphql.Resolver, rank *int) (interface{}, error) {

	var oktaToken = ctx.Value("jwtToken")
	if oktaToken == nil {
		return nil, fmt.Errorf("missing token in context")
	}
	verificationResult, err := ValidateRemotely(oktaToken.(string), os.Getenv("OKTA_CLIENT_ID_AP"), os.Getenv("OKTA_CLIENT_SECRET_AP"))
	fmt.Printf("Verification Result: %+v\n", verificationResult)
	if err != nil {
		// Set HTTP status to 403 Forbidden
		if w, ok := ctx.Value("http.ResponseWriter").(http.ResponseWriter); ok {
			w.WriteHeader(http.StatusForbidden)
		}
		graphql.AddError(ctx, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: "failed to validate token remotely",
			Extensions: map[string]interface{}{
				"code": "403",
			},
		})
		return nil, fmt.Errorf("failed to validate token remotely: %w", err)
	}
	if verificationResult["active"] != true {
		// Set HTTP status to 401 Unauthorized
		if w, ok := ctx.Value("http.ResponseWriter").(http.ResponseWriter); ok {
			w.WriteHeader(http.StatusUnauthorized)
		}
		graphql.AddError(ctx, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: "unauthorized: token is not active",
			Extensions: map[string]interface{}{
				"code": "401",
			},
		})
		return nil, fmt.Errorf("unauthorized: token is not active")
	}
	fmt.Println("Auth directive called")
	userRank := 0

	secretKey, err := core.GetSecretFromVault(os.Getenv("CLIENT_SECRET_AP"))
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}
	algorithm := os.Getenv("ALGORITHM")
	if algorithm == "" {
		algorithm = "HS256"
	}

	claims := jwt.MapClaims{}
	token, ok := ctx.Value("userToken").(string)
	if !ok || token == "" {
		return nil, fmt.Errorf("missing or invalid token")
	}

	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	fmt.Printf("Parsed Claims: %+v\n", claims)
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, fmt.Errorf("invalid token signature")
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !tkn.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	role := getClaimString(claims, "user_role", "userRole", "role")
	if role == "" {
		return nil, fmt.Errorf("role claim missing or invalid")
	}

	userIDStr := getClaimString(claims, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	userContext := model.UserContext{
		UserID:     userID,
		UserName:   getClaimString(claims, "userName", "user_name", "name", "displayName"),
		UserEmail:  getClaimString(claims, "userEmail", "user_email", "email", "preferred_username", "upn"),
		UserRole:   model.UserTypes(role),
		UserOffice: getClaimString(claims, "userOffice", "user_office", "office"),
		UserLob:    getClaimString(claims, "userLOB", "userLob", "user_lob", "lob"),
		UserGroups: getClaimIntSlice(claims, "userGroups", "user_groups", "groups"),
	}
	userContext.Rank = getRank(string(userContext.UserRole))

	ctx = context.WithValue(ctx, "user", userContext)
	fmt.Printf("User Object: %+v\n", userContext)

	userRank = getRank(role)
	fmt.Printf("User Rank: %d\n", userRank)
	if rank != nil && userRank < *rank {
		return nil, fmt.Errorf("insufficient permissions: required rank %d, user rank %d", *rank, userRank)
	}

	// implementation
	return next(ctx)
}
