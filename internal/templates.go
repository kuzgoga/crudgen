package internal

import (
	"strings"
)

type CrudTemplateContext struct {
	ServiceName string
	EntityType string
	EntityPlural string
}

var ServiceImports = []string{
	"app/internal/dal",
	"app/internal/models",
	//"errors"
	"gorm.io/gen/field",
	//"gorm.io/gorm"
}

var GetAllRawTemplate = `func (service *{{.ServiceName}}) GetAll() ([]*{{.EntityType}}, error) {
	var {{.EntityPlurar}} []*{{.EntityType}}
	{{.EntityPlural}}, err := dal.{{.EntityType}}.Preload(field.Associations).Find()
        return {{.EntityPlural}}, err
}`


func ToPlural(entityName string) string {
	return strings.ToLower(entityName) + "s"
}
