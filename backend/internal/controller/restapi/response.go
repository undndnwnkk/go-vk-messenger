package restapi

func NewTokenResponse(token string) tokenResponse {
	return tokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}
