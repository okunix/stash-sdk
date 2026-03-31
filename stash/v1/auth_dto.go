package stash

type GetTokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type GetTokenResponse struct {
	Token string `json:"token"`
}
