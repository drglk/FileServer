package entities

import "time"

type Document struct {
	ID        string    `db:"id"`
	OwnerID   string    `db:"owner_id"`
	Name      string    `db:"name"`
	Mime      string    `db:"mime"`
	IsFile    bool      `db:"is_file"`
	IsPublic  bool      `db:"is_public"`
	Path      string    `db:"path"`
	JSONData  []byte    `db:"json_data"`
	CreatedAt time.Time `db:"created_at"`
}
