package dto

type SessionRequest struct {
	Login    string `json:"login"`
	Password string `json:"pswd"`
}
