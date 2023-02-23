package in

type CollectIn struct {
	ComicId   string `json:"comicId" validate:"required" label:"漫画Id"`
	ChapterId string `json:"chapterId" label:"章节"`
	OrderId   int    `json:"orderId"`
	IsCollect int    `json:"isCollect" validate:"required" label:"收藏动作"`
}

type UserIsCollectComicIdIn struct {
	ComicId string `json:"comicId" validate:"required" label:"漫画Id"`
	OrderId int    `json:"orderId"`
}
