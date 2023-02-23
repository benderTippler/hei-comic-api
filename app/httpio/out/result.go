package out

type Result struct {
	Code    int         `json:"code"`
	Secret  interface{} `json:"secret"`
	IV      interface{} `json:"iv"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}
