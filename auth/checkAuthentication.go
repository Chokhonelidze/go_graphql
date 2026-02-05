

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	//"github.com/99designs/gqlgen/graphql"
	//"github.com/vektah/gqlparser/v2/gqlerror"
)

type UserContext struct {
	UserID      string
	Rank        int
	UserName    string
	UserEmail   string
	UserRole    string
	UserOffice  string
	UserLOB     string
	UserGroups  []int
}

// getRank returns the rank level based on user role
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

// // CheckAuthentication is a middleware that validates user authentication and authorization
// // rank: required rank level to access the resolver
// // role: optional role requirement
// func CheckAuthentication(requiredRank int, role string) func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
// 	return func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
// 		userRank := requiredRank
// 		userLOB := role

// 		// Get authorization header
// 		authHeader := graphql.GetFieldContext(ctx).Field.Name
// 		reqCtx := ctx.Value("request")
// 		if reqCtx == nil {
// 			return nil, &gqlerror.Error{
// 				Message: "Token Is Missing",
// 				Extensions: map[string]interface{}{
// 					"code": "403",
// 				},
// 			}
// 		}

// 		// Extract token from authorization header (assuming it comes through context)
// 		token := extractToken(ctx)
// 		if token == "" {
// 			return nil, &gqlerror.Error{
// 				Message: "Token Is Missing",
// 				Extensions: map[string]interface{}{
// 					"code": "403",
// 				},
// 			}
// 		}

// 		// Decode JWT token
// 		jwt, err := main.DecodeJWT(token)
// 		if err != nil || jwt == nil {
// 			return nil, &gqlerror.Error{
// 				Message: "Bad Or Expired Token",
// 				Extensions: map[string]interface{}{
// 					"code": "401",
// 				},
// 			}
// 		}

// 		// Get user token from headers
// 		userToken := extractUserToken(ctx)
// 		userInfo, err := main.CheckAccessToken(userToken)

// 		// Initialize user context with default values
// 		userCtx := &UserContext{
// 			UserID:     "",
// 			Rank:       0,
// 			UserName:   "",
// 			UserEmail:  jwt["email"].(string),
// 			UserRole:   "",
// 			UserOffice: "",
// 			UserLOB:    "legal",
// 			UserGroups: []int{},
// 		}

// 		if userInfo == nil {
// 			// User not authenticated
// 			if userRank == 0 {
// 				ctx = context.WithValue(ctx, "user", userCtx)
// 				return next(ctx)
// 			}
// 			return nil, &gqlerror.Error{
// 				Message: "Unauthorized",
// 				Extensions: map[string]interface{}{
// 					"code": "401",
// 				},
// 			}
// 		}

// 		// Populate user context from user info
// 		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
// 			userCtx.UserID = userInfoMap["user_id"].(string)
// 			userCtx.Rank = getRank(userInfoMap["user_role"].(string))
// 			userCtx.UserName = userInfoMap["user_name"].(string)
// 			userCtx.UserEmail = userInfoMap["user_email"].(string)
// 			userCtx.UserRole = userInfoMap["user_role"].(string)
// 			userCtx.UserLOB = "legal"

// 			if groups, ok := userInfoMap["doc_intel_group_ids"].([]int); ok {
// 				userCtx.UserGroups = groups
// 			}
// 		} else {
// 			return nil, &gqlerror.Error{
// 				Message: "Unauthorized",
// 				Extensions: map[string]interface{}{
// 					"code": "401",
// 				},
// 			}
// 		}

// 		// Check if user has required rank
// 		if getRank(userCtx.UserRole) >= userRank {
// 			ctx = context.WithValue(ctx, "user", userCtx)
// 			return next(ctx)
// 		}

// 		return nil, &gqlerror.Error{
// 			Message: "Unauthorized",
// 			Extensions: map[string]interface{}{
// 				"code": "401",
// 			},
// 		}
// 	}
// }

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
			// Extract JWT token from Authorization header
			authHeader := r.Header.Get("Authorization")
			token := ""
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}
			
			// if token == "" {
			// 	w.WriteHeader(http.StatusForbidden)
			// 	w.Write([]byte(`{"errors":[{"message":"Authorization header is missing or invalid","extensions":{"code":"403"}}]}`))
			// 	return
			// }
			fmt.Printf("Extracted Token: %s\n", token)

			// Extract user token
			userToken := r.Header.Get("User_token")
			// if userToken == "" {
			// 	w.WriteHeader(http.StatusForbidden)
			// 	w.Write([]byte(`{"errors":[{"message":"User_token header is missing","extensions":{"code":"403"}}]}`))
			// 	return
			// }
			fmt.Printf("Extracted User Token: %s\n", userToken)
			
			// Store tokens in context for use by resolvers
			ctx := context.WithValue(r.Context(), "jwtToken", token)
			ctx = context.WithValue(ctx, "userToken", userToken)
			
			// Continue with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
               
