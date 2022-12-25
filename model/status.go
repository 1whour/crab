package model

type StatusRequest struct {
	ID     string `form:"id" query:"id" json:"id"`
	Format string `json:"format" form:"format" query:"format"`
	// []string{key1, val1, key2, val2}
	Limit    int64  `json:"limit" form:"limit" query:"limit"`
	StartKey string `query:"start_key" form:"start_key" json:"start_key"`
	Sort     string
}
