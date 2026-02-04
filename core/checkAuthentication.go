

package core

import (
	"context"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"app/main"
	""
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

// CheckAuthentication is a middleware that validates user authentication and authorization
// rank: required rank level to access the resolver
// role: optional role requirement
func CheckAuthentication(requiredRank int, role string) func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		userRank := requiredRank
		userLOB := role

		// Get authorization header
		authHeader := graphql.GetFieldContext(ctx).Field.Name
		reqCtx := ctx.Value("request")
		if reqCtx == nil {
			return nil, &gqlerror.Error{
				Message: "Token Is Missing",
				Extensions: map[string]interface{}{
					"code": "403",
				},
			}
		}

		// Extract token from authorization header (assuming it comes through context)
		token := extractToken(ctx)
		if token == "" {
			return nil, &gqlerror.Error{
				Message: "Token Is Missing",
				Extensions: map[string]interface{}{
					"code": "403",
				},
			}
		}

		// Decode JWT token
		jwt, err := main.DecodeJWT(token)
		if err != nil || jwt == nil {
			return nil, &gqlerror.Error{
				Message: "Bad Or Expired Token",
				Extensions: map[string]interface{}{
					"code": "401",
				},
			}
		}

		// Get user token from headers
		userToken := extractUserToken(ctx)
		userInfo, err := main.CheckAccessToken(userToken)

		// Initialize user context with default values
		userCtx := &UserContext{
			UserID:     "",
			Rank:       0,
			UserName:   "",
			UserEmail:  jwt["email"].(string),
			UserRole:   "",
			UserOffice: "",
			UserLOB:    "legal",
			UserGroups: []int{},
		}

		if userInfo == nil {
			// User not authenticated
			if userRank == 0 {
				ctx = context.WithValue(ctx, "user", userCtx)
				return next(ctx)
			}
			return nil, &gqlerror.Error{
				Message: "Unauthorized",
				Extensions: map[string]interface{}{
					"code": "401",
				},
			}
		}

		// Populate user context from user info
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			userCtx.UserID = userInfoMap["user_id"].(string)
			userCtx.Rank = getRank(userInfoMap["user_role"].(string))
			userCtx.UserName = userInfoMap["user_name"].(string)
			userCtx.UserEmail = userInfoMap["user_email"].(string)
			userCtx.UserRole = userInfoMap["user_role"].(string)
			userCtx.UserLOB = "legal"

			if groups, ok := userInfoMap["doc_intel_group_ids"].([]int); ok {
				userCtx.UserGroups = groups
			}
		} else {
			return nil, &gqlerror.Error{
				Message: "Unauthorized",
				Extensions: map[string]interface{}{
					"code": "401",
				},
			}
		}

		// Check if user has required rank
		if getRank(userCtx.UserRole) >= userRank {
			ctx = context.WithValue(ctx, "user", userCtx)
			return next(ctx)
		}

		return nil, &gqlerror.Error{
			Message: "Unauthorized",
			Extensions: map[string]interface{}{
				"code": "401",
			},
		}
	}
}

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
               
