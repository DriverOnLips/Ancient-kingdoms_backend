package responseModels

type ResponseDefault struct {
	Code    int         `json:"Code"`
	Status  string      `json:"Status"`
	Message string      `json:"Message"`
	Body    interface{} `json:"Body"`
}
