package model

type Lambda struct {
	Funcs []Func `yaml:"func" json:"func"`
}

type Func struct {
	Name string `yaml:"name" json:"name"`
	Args string `yaml:"args" json:"args"`
}
