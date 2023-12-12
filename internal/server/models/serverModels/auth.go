package serverModels

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginResponce struct {
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type RegisterRequest struct {
	Name     string `json:"name"` // лучше назвать то же самое что login
	Password string `json:"password"`
}

type RegisterResponce struct {
	Status bool `json:"status"`
}
