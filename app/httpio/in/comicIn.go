package in

type ComicList struct {
	Name      string `json:"name"`
	Catalogue string `json:"catalogue"`
	Language  string `json:"language"`
	State     int    `json:"state"`
	Page      int    `json:"page"`
}

type ComparisonComic struct {
	Name  string `json:"name"`
	State int    `json:"state"`
	Page  int    `json:"page"`
}
