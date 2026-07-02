

package auth

import (
	"context"
	// "fmt"
	"net/http"
	// "net/url"
	"strings"
	// "app/core"
	// "encoding/json"
	// "io"
	"os"
)
 



// extractToken extracts the JWT token from the request context
func extractToken(ctx context.Context) string {
	// This depends on how your request headers are passed through context
	// Typically through *http.Request stored in context
	if req, ok := ctx.Value("request").(*http.Request); ok {
		authHeader := req.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				return parts[1]
			}
		}
	}
	return ""
}

// extractUserToken extracts the user token from the request context
func extractUserToken(ctx context.Context) string {
	if req, ok := ctx.Value("request").(*http.Request); ok {
		return req.Header.Get("User_token")
	}
	return ""
}

// HeaderCheckMiddleware validates required headers and extracts tokens
// This is an HTTP middleware that wraps the GraphQL handler
func HeaderCheckMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			allowedOrigins := parseAllowedOrigins(os.Getenv("FRONT_END_ADDRESSES"))
			if len(allowedOrigins) == 0 {
				allowedOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
			}
			if origin != "" {
				isAllowed := false
				normalizedOrigin := normalizeOrigin(origin)
				for _, o := range allowedOrigins {
					if normalizeOrigin(o) == normalizedOrigin {
						isAllowed = true
						break
					}
				}

				if !isAllowed {
					http.Error(w, "Unauthorized Origin", http.StatusForbidden)
					return // STOP execution here
				}
			} else {
				http.Error(w, "Origin header is required", http.StatusBadRequest)
				return
			}

			// Extract JWT token from Authorization header
			authHeader := r.Header.Get("Authorization")
			token := ""
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}


			//fmt.Printf("Extracted Token: %s\n", token)

			// Extract user token
			userToken := r.Header.Get("User_token")

			//fmt.Printf("Extracted User Token: %s\n", userToken)

			// Store tokens and response writer in context for use by resolvers and directives
			ctx := context.WithValue(r.Context(), "jwtToken", token)
			ctx = context.WithValue(ctx, "userToken", userToken)
			ctx = context.WithValue(ctx, "http.ResponseWriter", w)

			// Continue with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseAllowedOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		origins = append(origins, origin)
	}

	return origins
}

func normalizeOrigin(origin string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(origin)), "/")
}
               
// func ValidateRemotely(token, clientID, clientSecret string) (map[string]interface{}, error) {
// 	headers := map[string]string{
// 		"Accept":       "application/json",
// 		"Content-Type": "application/x-www-form-urlencoded",
// 	}

// 	data := url.Values{}
// 	data.Set("client_id", clientID)
// 	data.Set("client_secret", clientSecret)
// 	data.Set("token", token)

// 	introspectURL := core.OktaSettings.IntrospectURI

// 	req, err := http.NewRequest("POST", introspectURL, strings.NewReader(data.Encode()))
// 	if err != nil {
// 		return nil, err
// 	}

// 	for key, value := range headers {
// 		req.Header.Set(key, value)
// 	}

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if resp.StatusCode != 200 {
// 		return nil, fmt.Errorf("introspection returned status %d", resp.StatusCode)
// 	}

// 	var result map[string]interface{}
// 	err = json.Unmarshal(body, &result)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return result, nil
// }