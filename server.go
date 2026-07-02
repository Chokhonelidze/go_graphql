package main

import (
	"app/graph"
	"app/graph/directives"
	"app/graph/generated"
	"app/graph/model"
	migrate "app/migrations"

	//"app/core"
	"app/auth"
	// "crypto/tls"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/rs/cors"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	// "github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/vektah/gqlparser/v2/ast"
	// "gorm.io/driver/postgres"
	// "gorm.io/gorm"
	"app/core"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

const defaultPort = "8080"

func main() {
	router := chi.NewRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	allowedOrigins := parseAllowedOrigins(getEnv("FRONT_END_ADDRESSES", "http://localhost:3000,http://localhost:8080"))
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"https://localhost:3000", "http://localhost:5000"}
	}
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "access-control-allow-headers", "access-control-allow-origin", "", "Accept", "Authorization", "Content-Type", "X-CSRF-Token", "User-Token", "User_Token"},
		ExposedHeaders:   []string{"Content-Length", "X-JSON-Response"},
		MaxAge:           3600,
		Debug:            true,
	}).Handler)

	db, err := core.PostgresConnection()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get database instance: %v", err)
	}

	defer sqlDB.Close()
	// ctx := context.Background()
	// client, cosmosContainers, err := core.InitializeCosmos(ctx)
	// if err != nil {
	// 	log.Fatalf("failed to initialize Cosmos DB: %v", err)
	// }
	// fmt.Printf("Cosmos Client: %v\n", client)
	// fmt.Printf("Cosmos Containers: %v\n", cosmosContainers)
	httpClient := &http.Client{
		Timeout: 90 * time.Second,
	}

	resolvers := &graph.Resolver{
		DB: db,
		//Container: cosmosContainers,
		HTTPClient: httpClient,
	}

	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolvers, Directives: generated.DirectiveRoot{
		Auth: directives.Auth,
	}}))

	if getEnvBool("USE_GORM_AUTOMIGRATE", false) {
		if err := db.AutoMigrate(&model.User{}, &model.Songs{}, &model.Song{}, &model.Downloads{}); err != nil {
			log.Fatalf("failed to run gorm auto-migrate: %v", err)
		}
		log.Println("Checking CSV data imports for Songs...")
		var count int64
		if err := db.Model(&model.Songs{}).Count(&count).Error; err != nil {
			log.Fatalf("failed to check songs table: %v", err)
		}
		if count == 0 {
			if err := model.MigrateSongs(db); err != nil {
				log.Fatalf("failed to seed songs from CSV: %v", err)
			}
		} else {
			log.Println("Songs data already exists in the database. Skipping CSV import.")
		}
		log.Println("Songs database verification completed.")

		var countSong int64
		if err := db.Model(&model.Song{}).Count(&countSong).Error; err != nil {
			log.Fatalf("failed to check song table: %v", err)
		}
		if countSong == 0 {
			log.Println("Song table is empty. Running migration for Song table...")
			if err := model.MigrateSong(db); err != nil {
				log.Fatalf("failed to migrate Song table: %v", err)
			} else {
				log.Println("Song table migration completed successfully.")
			}
		} else {
			log.Println("Song table already exists in the database. Skipping migration.")
		}

	} else {
		if err := migrate.RunMigrations(os.Getenv("DB_CONNECTION_STRING")); err != nil {
			log.Fatalf("failed to apply sql migrations: %v", err)
		}
	}

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})
	srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
		if gqlErr, ok := err.(*gqlerror.Error); ok {
			code := "INTERNAL_ERROR"
			if ext, ok := gqlErr.Extensions["code"]; ok {
				code = ext.(string)
			}
			return &gqlerror.Error{
				Message:   gqlErr.Message,
				Path:      gqlErr.Path,
				Locations: gqlErr.Locations,
				Extensions: map[string]interface{}{
					"code": code,
				},
			}
		}
		return gqlerror.Errorf("internal server error")
	})
	router.Handle("/", ApolloSandboxHandler())
	router.Handle("/graphql", auth.HeaderCheckMiddleware()(srv))
	// http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
	// 	if r.Method == http.MethodPost {
	// 		auth.HeaderCheckMiddleware()(srv).ServeHTTP(w, r)
	// 	} else {
	// 		srv.ServeHTTP(w, r)
	// 	}
	// })

	serverError := http.ListenAndServe(":8080", router)
	if serverError != nil {
		log.Fatalf("failed to start server: %v", serverError)
	}

	// log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func ApolloSandboxHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		// if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		// 	scheme = "https"
		// }

		host := r.Host
		// if !strings.Contains(host, "localhost") {
		// 	scheme = "https"
		// }

		fullEndpoint := fmt.Sprintf("%s://%s/graphql", scheme, host)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w,
			`<div style='width: 100%%; height: 100vh;' id='embedded-sandbox'></div>
            <script src='https://embeddable-sandbox.cdn.apollographql.com/_latest/embeddable-sandbox.umd.production.min.js'></script> 
            <script>
            new window.EmbeddedSandbox({
                target: '#embedded-sandbox',
                initialEndpoint: '%s',
                includeCookies: false,
            });
            </script>
        `, fullEndpoint)
	}
}
