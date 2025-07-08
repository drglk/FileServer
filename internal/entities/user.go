package entities

type User struct {
	ID       string `db:"id"`
	Login    string `db:"login"`
	PassHash []byte `db:"pass_hash"`
}
