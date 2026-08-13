package dto

type LoginRequest struct {
	Username string `json:"username" minLength:"1" maxLength:"64"`
	Password string `json:"password" minLength:"1" maxLength:"72"`
}

type LoginUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type LoginResponse struct {
	AccessToken string    `json:"accessToken"`
	TokenType   string    `json:"tokenType" enum:"Bearer"`
	ExpiresIn   int64     `json:"expiresIn"`
	User        LoginUser `json:"user"`
}
