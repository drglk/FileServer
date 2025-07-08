package dto

type UserRequest struct {
	AdminToken string `json:"token"`
	Login      string `json:"login"`
	Password   string `json:"pswd"`
}
