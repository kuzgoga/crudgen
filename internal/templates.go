package internal

type CrudTemplatesContext struct {
	ServiceName  string
	EntityType   string
	EntityPlural string
}

var ServiceImports = []string{
	"app/internal/dal",
	"app/internal/models",
	"errors",
	"gorm.io/gen/field",
	"gorm.io/gorm",
}

const CreateMethod = "Create"
const GetAllMethod = "GetAll"
const GetByIdMethod = "GetById"
const UpdateMethod = "Update"
const DeleteMethod = "Delete"
const CountMethod = "Count"

var RawTemplates = map[string]string{
	CreateMethod:  CreateRawTemplate,
	GetAllMethod:  GetAllRawTemplate,
	GetByIdMethod: GetByIdRawTemplate,
	UpdateMethod:  UpdateRawTemplate,
	DeleteMethod:  DeleteRawTemplate,
	CountMethod:   CountRawTemplate,
}
