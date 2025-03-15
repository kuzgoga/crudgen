package internal

const CreateRawTemplate = `func (service *{{.ServiceName}}) Create(item {{.EntityType}}) ({{.EntityType}}, error) {
    utils.ReplaceEmptySlicesWithNil(&item)
    err := dal.{{.EntityType}}.Create(&item)
    if err != nil {
        return item, err
	}
    err = utils.AppendAssociations(database.GetInstance(), &item)
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
    utils.ReplaceEmptySlicesWithNil(&item)
    
    _, err := dal.Author.Updates(&item)
	if err != nil {
		return item, err
	}
    
    err = utils.UpdateAssociations(database.GetInstance(), &item)

    if err != nil {
		return item, err
	}

    return item, err
}`

const DeleteRawTemplate = `func (service *{{.ServiceName}}) Delete(id uint) error {
    _, err := dal.{{.EntityType}}.Unscoped().Where(dal.{{.EntityType}}.Id.Eq(id)).Delete()
    return err
}`

const CountRawTemplate = `func (service *{{.ServiceName}}) Count() (int64, error) {
    amount, err := dal.{{.EntityType}}.Count()
    return amount, err
}`

const SortedByOrderTemplate = `func (service *{{.ServiceName}}) SortedByOrder(fieldsSortingOrder []utils.SortField) ([]*{{.EntityType}}, error) {
	return utils.SortByOrder(fieldsSortingOrder, {{.EntityType}}{})
}`

const SearchByAllStringFields = `func (service *{{.ServiceName}}) SearchByAllTextFields(phrase string) ([]*{{.EntityType}}, error) {
	return utils.FindPhraseByStringFields[{{.EntityType}}](phrase, {{.EntityType}}{})
}`

var implementedMethods = []string{CreateMethod, GetAllMethod, GetByIdMethod, UpdateMethod, DeleteMethod, CountMethod, SortedByOrderMethod, SearchByAllStringFields}
