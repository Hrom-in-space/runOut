package domain

type Audio struct {
	Format string
	Data   []byte
}

type Need struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}
