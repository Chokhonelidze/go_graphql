package helperFunctions

import (
    "app/graph/model"
    // "encoding/base64"
    "fmt"
    "reflect"
    "strings"
    "os"
    "time"
    "context"
    "app/core"
    "github.com/xuri/excelize/v2"
    "github.com/google/uuid"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"

)


func formatHeaderText(header string) string {
    header = strings.ReplaceAll(header, "_", " ")
    header = strings.Title(header)
    return header
}

func DownloadExcel(data interface{}, fields []string, object model.Tables) (string, error) {
    v := reflect.ValueOf(data)
    if v.Kind() != reflect.Slice {
        return "", fmt.Errorf("data must be a slice")
    }

    headers := normalizeFields(fields)
    if len(headers) == 0 {
        headers = inferHeaders(v)
    }
    if len(headers) == 0 {
        return "", fmt.Errorf("no fields available for export")
    }

    f := excelize.NewFile()
    defaultSheet := f.GetSheetName(f.GetActiveSheetIndex())
    if defaultSheet == "" {
        defaultSheet = string(object)
    }

    // Write headers
    for colIdx, header := range headers {
        cell, err := excelize.CoordinatesToCellName(colIdx+1, 1)
        if err != nil {
            return "", fmt.Errorf("failed to calculate header cell: %w", err)
        }
        f.SetCellValue(defaultSheet, cell, formatHeaderText(header))
    }
  
   
    // Write data rows from GORM query result
    for rowIdx := 0; rowIdx < v.Len(); rowIdx++ {
        item := v.Index(rowIdx)
        for item.Kind() == reflect.Ptr {
            if item.IsNil() {
                break
            }
            item = item.Elem()
        }

        for colIdx, header := range headers {
            cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
            if err != nil {
                return "", fmt.Errorf("failed to calculate data cell: %w", err)
            }
            fmt.Println("Resolving value for header:", header)
            fmt.Printf("Row %d, Column %d (Header: %s)\n", rowIdx+2, colIdx+1, header)
            fmt.Printf("Item: %+v\n", item.Interface())
            f.SetCellValue(defaultSheet, cell, resolveFieldValue(item, header))
        }
    }
    buffer, err := f.WriteToBuffer()
    if err != nil {
        return "", fmt.Errorf("failed to generate Excel file: %w", err)
    }

    accountName := os.Getenv("AZURE_STORAGE_ACCT")
	containerName := "processed"
	// blobPath := fmt.Sprintf("landing/excel/%s", object+".xlsx")
	accountKeyName := os.Getenv("AZURE_ACCOUNT_KEY_NAME")
	accountKey, err := core.GetSecretFromVault(accountKeyName)
	if err != nil {
		return "", fmt.Errorf("failed to get account key from vault: %w", err)
	}

       // 1. Create the Shared Key Credential
    credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
    if err != nil {
        return "", fmt.Errorf("failed to create Azure credential: %w", err)
    }

    // 2. Create a Service Client
    serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
    client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
    if err != nil {
        return "", fmt.Errorf("failed to create Azure service client: %w", err)
    }

    // 3. Define your blob path (ensure it doesn't start with a slash)
   
    
    randomGeneratedUUID := uuid.New()

    blobName := fmt.Sprintf("exports/%s/%s.xlsx", randomGeneratedUUID, object)
    
    // 4. Upload the buffer directly via the client
    _, err = client.UploadBuffer(context.TODO(), containerName, blobName, buffer.Bytes(), &azblob.UploadBufferOptions{})
    if err != nil {
        return "", fmt.Errorf("failed to upload Excel file: %w", err)
    }

    // 5. Generate SAS Token
    expiryTime := time.Now().UTC().Add(1 * time.Hour)
    
    // Note: Version "2023-11-03" or similar is often required by the signer
    sasValues := sas.BlobSignatureValues{
        Protocol:      sas.ProtocolHTTPS,
        StartTime:     time.Now().UTC().Add(-5 * time.Minute), // Buffer for clock skew
        ExpiryTime:    expiryTime,
        Permissions:   "r", // 'r' for read
        ContainerName: containerName,
        BlobName:      blobName,
    }

    sasQueryParams, err := sasValues.SignWithSharedKey(credential)
    if err != nil {
        return "", fmt.Errorf("failed to generate SAS token: %w", err)
    }

    // 6. Construct the final URL
    sasURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
        accountName, containerName, blobName, sasQueryParams.Encode())
    return sasURL, nil
}

func normalizeFields(fields []string) []string {
    result := make([]string, 0, len(fields))
    seen := make(map[string]struct{}, len(fields))

    for _, field := range fields {
        if field == "" {
            continue
        }
        name := strings.TrimSpace(field)
        if name == "" {
            continue
        }
        if _, ok := seen[name]; ok {
            continue
        }
        seen[name] = struct{}{}
        result = append(result, name)
    }

    return result
}

func inferHeaders(v reflect.Value) []string {
    if v.Len() == 0 {
        return nil
    }

    item := v.Index(0)
    for item.Kind() == reflect.Ptr {
        if item.IsNil() {
            return nil
        }
        item = item.Elem()
    }
    if item.Kind() != reflect.Struct {
        return nil
    }

    t := item.Type()
    headers := make([]string, 0, t.NumField())
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if field.PkgPath != "" {
            continue
        }
        headers = append(headers, field.Name)
    }

    return headers
}

func resolveFieldValue(item reflect.Value, header string) interface{} {
    fmt.Printf("Resolving field value for header: %s\n", header)    
    fmt.Printf("Current item: %+v\n", item)
    if strings.EqualFold(header, "id") {
        header = "ID"
        var field reflect.Value
        field = item.FieldByName("Model").FieldByName("ID")
        return field.Interface()
    }
    
    if !item.IsValid() || item.Kind() != reflect.Struct {
        return ""
    }

    t := item.Type()
    var field reflect.Value

    // Iterate through struct fields to find a match
    for i := 0; i < t.NumField(); i++ {
        structField := t.Field(i)
        
        // Match by: 1. Field Name, 2. JSON tag, or 3. GORM column name
        if strings.EqualFold(structField.Name, header) || 
           strings.EqualFold(structField.Tag.Get("json"), header) || 
           strings.EqualFold(getGormColumnName(structField.Tag.Get("gorm")), header) {
            field = item.Field(i)
            break
        }
    }

    if !field.IsValid() {
        return ""
    }

    // Handle Pointers
    for field.Kind() == reflect.Ptr {
        if field.IsNil() {
            return ""
        }
        field = field.Elem()
    }

    if field.IsValid() && field.CanInterface() {
        return field.Interface()
    }

    return ""
}

// Helper to safely extract the column name from GORM tags
func getGormColumnName(tag string) string {
    if tag == "" {
        return ""
    }
    // GORM tags are semicolon separated: gorm:"column:user_id;primaryKey"
    settings := strings.Split(tag, ";")
    for _, setting := range settings {
        if strings.HasPrefix(strings.ToLower(setting), "column:") {
            parts := strings.Split(setting, ":")
            if len(parts) > 1 {
                return parts[1] // Safely return user_id
            }
        }
    }
    return ""
}