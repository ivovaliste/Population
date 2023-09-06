package models

type User struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	UserPhone string `json:"userphone"`
	Email     string `json:"email"`
}
