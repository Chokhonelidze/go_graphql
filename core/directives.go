package core

import (
	"context"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// AuthDirective validates user authentication and authorization
func AuthDirective(ctx context.Context, obj interface{}, next graphql.Resolver, rank int) (interface{}, error) {
	userCtx := ctx.Value("user").(*UserContext)
	if userCtx == nil {
		return nil, &gqlerror.Error{
			Message: "Unauthorized",
			Extensions: map[string]interface{}{
				"code": "401",
			},
		}
	}

	// Check if user has required rank
	if userCtx.Rank >= rank {
		return next(ctx)
	}

	return nil, &gqlerror.Error{
		Message: "Forbidden",
		Extensions: map[string]interface{}{
			"code": "403",
		},
	}
}

// AuthenticationMiddleware extracts and validates tokens from request headers
func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract token from Authorization header
		token := extractToken(r)
		if token == "" {
			ctx = context.WithValue(ctx, "user", &UserContext{
				UserID:     "",
				Rank:       0,
				UserName:   "",
				UserEmail:  "",
				UserRole:   "",
				UserOffice: "",
				UserLOB:    "",
				UserGroups: []int{},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Store token in context for later use
		ctx = context.WithValue(ctx, "token", token)
		ctx = context.WithValue(ctx, "request", r)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken extracts JWT token from Authorization header
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}
	return ""
}
