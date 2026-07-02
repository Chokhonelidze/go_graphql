package helperFunctions

import (
	"fmt"
	//"os"
	"app/graph/model"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"gorm.io/gorm"
)

func InputInterfaceWithCountry(input model.StandardInputInterfaceAndCountry) (int, int, []*model.Search, string, *model.MSSQLUUID, *string) {
	limit := input.StandardInput.Limit
	offset := input.StandardInput.Offset
	search := input.StandardInput.Search

	var limitInt int
	if limit != nil {
		limitInt = int(*limit)
	} else {
		limitInt = 10
	}
	var offsetInt int
	if offset != nil {
		offsetInt = int(*offset)
	} else {
		offsetInt = 0
	}
	sorts := input.StandardInput.Sorts
	insuredNameID := input.StandardInput.InsuredNameID
	country := input.Country
	var sort_list string
	for _, sort := range sorts {
		if sort_list != "" {
			sort_list += ", "
		}
		sort_list += *sort.Sort + " " + string(*sort.Direction)
	}
	return limitInt, offsetInt, search, sort_list, insuredNameID, country
}

func InputInterface(input model.StandardInputInterface) (int, int, []*model.Search, string, *model.MSSQLUUID) {
	limit := input.Limit
	offset := input.Offset
	search := input.Search
	var limitInt int
	if limit != nil {
		limitInt = int(*limit)
	} else {
		limitInt = 10
	}
	var offsetInt int
	if offset != nil {
		offsetInt = int(*offset)
	} else {
		offsetInt = 0
	}
	sorts := input.Sorts
	insuredNameID := input.InsuredNameID
	var sort_list string
	for _, sort := range sorts {
		if sort_list != "" {
			sort_list += ", "
		}
		sort_list += *sort.Sort + " " + string(*sort.Direction)
	}
	return limitInt, offsetInt, search, sort_list, insuredNameID
}

func InputFieldExtractor(fields []graphql.CollectedField, country *string, insuredNameID *model.MSSQLUUID, db *gorm.DB, tableType string) (fieldNames []string, countField bool, tableMessages []*model.TableMessage, preload bool) {
	for _, children := range fields {
		if children.Name == "count" {
			countField = true
			continue
		}

		if children.Name == "data" {
			for _, selection := range children.SelectionSet {
				// Type assert to *ast.Field to access the Name property
				if field, ok := selection.(*ast.Field); ok {
					if field.Name == "InsuredNames" {
						preload = true
						continue
					}
					fieldNames = append(fieldNames, field.Name)
				}
			}
		}
		if children.Name == "messages" {
			if country != nil {
				if err := db.Where("insured_name_id = ?", insuredNameID).Where("country = ?", country).Where("table_messages.[table] = ?", tableType).Find(&tableMessages).Error; err != nil {
					// handle error if needed
				}
			} else {
				if err := db.Where("insured_name_id = ?", insuredNameID).Where("table_messages.[table] = ?", tableType).Find(&tableMessages).Error; err != nil {
					// handle error if needed
				}
			}
			continue
		}
	}
	return fieldNames, countField, tableMessages, preload
}

func SearchBuilder(searches []*model.Search, query *gorm.DB) *gorm.DB {
	for _, search := range searches {
		var operation string
		if search.Operation != "" {
			operation = string(search.Operation)
		} else {
			operation = "EQUAL"
		}
		switch operation {
		case "NOT_EQUAL":
			query = query.Where(search.Field+" != ?", search.Search)
		case "EQUAL":
			query = query.Where(search.Field+" = ?", search.Search)
		case "CONTAINS":
			query = query.Where(search.Field+" LIKE ?", "%"+search.Search+"%")
		case "LIKE":
			query = query.Where(search.Field+" LIKE ?", search.Search+"%")
		case "TIME":
			var searchTime time.Time
			var err error
			formats := []string{time.RFC3339, "2006-01-02", "1/2/2006", "01/02/2006", "01/2/2006", "1/02/2006", "1/2/06", "01/02/06", "2006-1-2"}
			for _, format := range formats {
				searchTime, err = time.ParseInLocation(format, strings.TrimSpace(search.Search), time.Local)
				if err == nil {
					searchTime = searchTime.Truncate(time.Millisecond)
					break
				}
			}
			if err == nil {
				fmt.Printf("Parsed time successfully: %v\n", searchTime)
				nextDay := searchTime.AddDate(0, 0, 1)
				query = query.Where(search.Field+" >= ? AND "+search.Field+" < ?", searchTime, nextDay)
			}
		default:
			query = query.Where(search.Field+" = ?", search.Search)
		}
	}
	return query
}
