package out

type ChapterOut struct {
	Count    int        `json:"count"`
	Chapters []*Chapter `json:"chapters"`
}

type Chapter struct {
	UUID      string `json:"id"`        //雪花id
	Name      string `json:"name"`      // 章节名称
	Pid       int64  `json:"pid"`       // 主表漫画表
	Sort      int    `json:"sort"`      // 章节排序
	OrderId   int    `json:"orderId"`   //
	State     int    `json:"state"`     // 1、资源正常 2、资源损坏
	Cover     string `json:"cover"`     //章节默认封面
	ImageType string `json:"imageType"` //封面类型
}

type ChapterReadyResourceOut struct {
	Status   int `json:"status"`
	Progress int `json:"progress"`
}

type ChapterResourceOut struct {
	Resource  []string `json:"resource"`
	ImageType string   `json:"imageType"`
	Count     int      `json:"count"`
	AllPage   int      `json:"allPage"`
}

type Resource struct {
	Path      string `json:"path"`
	Sort      int    `json:"sort"`
	ChapterId int64  `json:"chapterId"`
	State     int    `json:"state"`
}
