package in

type ChapterReadyResource struct {
	ChapterId string `json:"chapterId"`
	OrderId   string `json:"orderId"`
}

type ChapterResource struct {
	ChapterId string `json:"chapterId"`
	OrderId   string `json:"orderId"`
	Page      int    `json:"page"`
}

type GetComicChapters struct {
	ComicId string `json:"comicId"`
	Page    int    `json:"page"`
}

type GetComicChapterTops struct {
	ComicId string `json:"comicId"`
}
