package out

import "hei-comic-api/app/model"

type FindDictOut struct {
	Dictionary []*model.Dictionary `json:"dictionary"`
}
