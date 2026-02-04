package main

import (
	"app/graph"
	"app/graph/model"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	// "github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/vektah/gqlparser/v2/ast"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	dbHost := getEnv("DB_HOST", "postgres_db")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "graphql_user")
	dbPassword := getEnv("DB_PASSWORD", "securepassword")
	dbName := getEnv("DB_NAME", "graphql_db")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get database instance: %v", err)
	}

	defer sqlDB.Close()


	resolvers := &graph.Resolver{DB: db}
	
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolvers}))
	 
	if err := db.AutoMigrate(&model.Student{}, &model.ZurichTeam{}, &model.Document{}); err != nil {
        log.Fatalf("failed to migrate database: %v", err)
    }

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	http.Handle("/", ApolloSandboxHandler())
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func ApolloSandboxHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}

		host := r.Host
		if !strings.Contains(host, "localhost") {
			scheme = "https"
		}

		fullEndpoint := fmt.Sprintf("%s://%s/query", scheme, host)

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
 