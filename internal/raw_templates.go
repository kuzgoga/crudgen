package internal

const CreateRawTemplate = `func (service *{{.ServiceName}}) Create(item {{.EntityType}}) ({{.EntityType}}, error) {
    err := dal.{{.EntityType}}.Preload(field.Associations).Create(&item)
    return item, err
}`

const GetAllRawTemplate = `func (service *{{.ServiceName}}) GetAll() ([]*{{.EntityType}}, error) {
    var {{.EntityPlural}} []*{{.EntityType}}
    {{.EntityPlural}}, err := dal.{{.EntityType}}.Preload(field.Associations).Find()
    return {{.EntityPlural}}, err
}`

const GetByIdRawTemplate = `func (service *{{.ServiceName}}) GetById(id uint) (*{{.EntityType}}, error) {
    item, err := dal.{{.EntityType}}.Preload(field.Associations).Where(dal.{{.EntityType}}.Id.Eq(id)).First()
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        } else {
            return nil, err
        }
    }
    return item, nil
}`

const UpdateRawTemplate = `func (service *{{.ServiceName}}) Update(item {{.EntityType}}) ({{.EntityType}}, error) {
    err := dal.{{.EntityType}}.Preload(field.Associations).Save(&item)
    return item, err
}`

const DeleteRawTemplate = `func (service *{{.ServiceName}}) Delete(item {{.EntityType}}) ({{.EntityType}}, error) {
    _, err := dal.{{.EntityType}}.Unscoped().Preload(field.Associations).Delete(&item)
    return item, err
}`

const CountRawTemplate = `func (service *{{.ServiceName}}) Count() (int64, error) {
    amount, err := dal.{{.EntityType}}.Count()
    return amount, err
}`
