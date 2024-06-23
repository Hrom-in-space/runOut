package models

type Need struct {
	ID   int    `json:"id" gorm:"primaryKey,autoIncrement"`
	Name string `json:"name"`
}
