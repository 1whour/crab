package gate

type Page struct {
	Limit int    `form:"limit"`
	Page  int    `form:"page"`
	Sort  string `form:"sort"`
}
