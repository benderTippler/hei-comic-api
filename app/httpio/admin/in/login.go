package in

type CapchaIn struct {
	Height int `json:"height" in:"query" default:"40"`
	Width  int `json:"width"  in:"query" default:"150"`
}
