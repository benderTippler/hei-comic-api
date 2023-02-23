package adapter

type Adapter struct {
	Name         string   `yaml:"name"`
	OrderId      int      `yaml:"orderId"`
	RealTime     bool     `yaml:"realTime"`
	Sort         int      `yaml:"sort"`
	ComicChan    int      `yaml:"comicChan"`
	ChapterChan  int      `yaml:"chapterChan"`
	Origin       string   `yaml:"origin"`
	Referer      string   `yaml:"referer"`
	MaxTry       int      `yaml:"maxTry"`
	Language     string   `yaml:"language"`
	Switch       bool     `yaml:"switch"`
	CachePath    string   `yaml:"cachePath"`
	Scope        []string `yaml:"scope"`
	ListTemplate []string `yaml:"listTemplate"`
	IsPage       bool     `yaml:"isPage"`
	State        []int    `yaml:"state"`
}
