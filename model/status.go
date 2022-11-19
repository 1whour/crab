package model

type StatusRequest struct {
	Format string
	// []string{key1, val1, key2, val2}
	Filter []string
}
