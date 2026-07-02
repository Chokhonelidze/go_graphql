package core

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

// GetSecretFromVault retrieves a secret value from Azure Key Vault
func GetSecretFromVault(secretName string) (string, error) {
	// Log the secret name being retrieved
	fmt.Printf("Attempting to retrieve secret: %s\n", secretName)

	// Create DefaultAzureCredential with additional allowed tenants
	opts := &azidentity.DefaultAzureCredentialOptions{
		AdditionallyAllowedTenants: []string{"*"},
	}
	credential, err := azidentity.NewDefaultAzureCredential(opts)
	if err != nil {
		return "", fmt.Errorf("failed to create credential: %w", err)
	}

	// Get Key Vault name from environment variable
	keyVaultName := os.Getenv("AZURE_KEYVAULT_NAME")
	if keyVaultName == "" {
		return "", fmt.Errorf("AZURE_KEYVAULT_NAME environment variable is not set")
	}

	keyVaultURI := fmt.Sprintf("https://%s.vault.azure.net", keyVaultName)

	// Create Secret Client
	client, err := azsecrets.NewClient(keyVaultURI, credential, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create secret client: %w", err)
	}

	// Get the secret
	if secretName == "" {
		return "", fmt.Errorf("secret name cannot be empty")
	}
	retrievedSecret, err := client.GetSecret(context.Background(), secretName, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Return the secret value
	return *retrievedSecret.Value, nil
}
