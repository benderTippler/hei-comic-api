package in

type CreateDictIn struct {
	Field   string `json:"field" validate:"required" label:"字段"`
	Content string `json:"content" validate:"required" label:"值"`
}
