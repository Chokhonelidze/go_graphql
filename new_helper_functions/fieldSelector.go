package new_helper_functions

import (
	ast "github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

func OnlyFieldNames(fields []graphql.CollectedField) []string {
	var fieldNames []string
	for _, children := range fields {
		if children.Name == "count" || children.Name == "success" || children.Name == "errors" {
			continue
		}
		if children.Name == "data" {
			for _, selection := range children.SelectionSet {
				// Type assert to *ast.Field to access the Name property
				if field, ok := selection.(*ast.Field); ok {
					fieldNames = append(fieldNames, field.Name)
				}
			}
		}
	}
	return fieldNames
}
