package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

const (
	DatabaseName  = "testDatabase"
	ContainerName = "container"
)

// CosmosContainers holds references to all Cosmos DB containers
type CosmosContainers struct {
	Container *azcosmos.ContainerClient
}

// CreateCosmosClient creates and returns a Cosmos DB client
func CreateCosmosClient(ctx context.Context) (*azcosmos.Client, error) {
	cosmosURLName := os.Getenv("COSMOS_URL_NAME")
	cosmosKeyName := os.Getenv("COSMOS_KEY_NAME")

	if cosmosURLName == "" || cosmosKeyName == "" {
		return nil, fmt.Errorf("COSMOS_URL_NAME and COSMOS_KEY_NAME environment variables must be set")
	}

	url, err := GetSecretFromVault(cosmosURLName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cosmos URL from vault: %w", err)
	}

	key, err := GetSecretFromVault(cosmosKeyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cosmos Key from vault: %w", err)
	}

	log.Printf("cosmos URL retrieved: %s\n", url)
	log.Println("cosmos Key retrieved successfully")

	cred, err := azcosmos.NewKeyCredential(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	clientOptions := &azcosmos.ClientOptions{}

	client, err := azcosmos.NewClientWithKey(url, cred, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cosmos client: %w", err)
	}

	return client, nil
}

// GetOrCreateDatabase gets an existing database or creates a new one
func GetOrCreateDatabase(ctx context.Context, client *azcosmos.Client, databaseName string) (*azcosmos.DatabaseClient, error) {
	databaseClient, err := client.NewDatabase(databaseName)

	_, readErr := databaseClient.Read(ctx, nil)
	if readErr == nil {
		log.Printf("Database '%s' already exists\n", databaseName)
		return databaseClient, nil
	}

	log.Printf("Creating database '%s'\n", databaseName)
	databaseProperties := azcosmos.DatabaseProperties{
		ID: databaseName,
	}

	_, err = client.CreateDatabase(ctx, databaseProperties, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return databaseClient, nil
}

// GetOrCreateContainer gets an existing container or creates a new one
func GetOrCreateContainer(ctx context.Context, database *azcosmos.DatabaseClient, containerName string, partitionKeyPath string, throughput int32) (*azcosmos.ContainerClient, error) {
	containerClient, err := database.NewContainer(containerName)

	_, readErr := containerClient.Read(ctx, nil)
	if readErr == nil {
		log.Printf("Container '%s' already exists\n", containerName)
		return containerClient, nil
	}

	log.Printf("Creating container '%s' with partition key '%s'\n", containerName, partitionKeyPath)

	containerProperties := azcosmos.ContainerProperties{
		ID: containerName,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{partitionKeyPath},
		},
	}

	throughputProperties := azcosmos.NewManualThroughputProperties(throughput)

	_, err = database.CreateContainer(ctx, containerProperties, &azcosmos.CreateContainerOptions{
		ThroughputProperties: &throughputProperties,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return containerClient, nil
}

// CosmosSetup initializes all Cosmos DB resources
func CosmosSetup(ctx context.Context, client *azcosmos.Client) (*CosmosContainers, error) {
	log.Println("Starting Cosmos DB setup...")

	database, err := GetOrCreateDatabase(ctx, client, DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	container, err := GetOrCreateContainer(ctx, database, ContainerName, "/id", 400)
	if err != nil {
		return nil, fmt.Errorf("failed to setup container: %w", err)
	}

	log.Println("Cosmos DB setup complete!")

	return &CosmosContainers{
		Container: container,
	}, nil
}

// SetupCosmosForDevelopment sets up Cosmos client for development with emulator
func SetupCosmosForDevelopment(connectionString string) (*azcosmos.Client, error) {
	client, err := azcosmos.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cosmos client from connection string: %w", err)
	}

	return client, nil
}



// InitializeCosmos initializes Cosmos DB client based on environment (development or production)
func InitializeCosmos(ctx context.Context) (*azcosmos.Client, *CosmosContainers, error) {
	connectionString := os.Getenv("COSMOS_CONNECTION_STRING")

	var client *azcosmos.Client
	var err error

	if connectionString != "" {
		// Development mode with connection string
		log.Println("Using Cosmos DB connection string (development mode)")

		maxRetries := 30
		retryDelay := 5 * time.Second

		log.Println("Connecting to Cosmos DB emulator...")
		for i := 0; i < maxRetries; i++ {
			client, err = azcosmos.NewClientFromConnectionString(connectionString, nil)
			if err == nil {
				log.Println("Successfully connected to Cosmos DB")
				break
			}

			if i < maxRetries-1 {
				log.Printf("Cosmos DB not ready (attempt %d/%d), retrying in %v...\n", i+1, maxRetries, retryDelay)
				time.Sleep(retryDelay)
			} else {
				return nil, nil, fmt.Errorf("failed to connect to Cosmos DB after %d attempts: %w", maxRetries, err)
			}
		}
	} else {
		// Production mode with Key Vault
		log.Println("Using Azure Key Vault for Cosmos DB credentials (production mode)")
		client, err = CreateCosmosClient(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Cosmos client: %w", err)
		}
	}

	// Setup Cosmos DB databases and containers
	cosmosContainers, err := CosmosSetup(ctx, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup Cosmos DB: %w", err)
	}

	return client, cosmosContainers, nil
}