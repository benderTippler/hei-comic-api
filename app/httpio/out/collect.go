package out

import (
	"hei-comic-api/app/model/mongo"
)

type UserCollectsOut struct {
	Comics []*mongo.Comic `json:"comics"`
}

type UserIsCollectComicId struct {
	CollectType int `json:"collectType"`
}
