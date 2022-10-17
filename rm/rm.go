package rm

type Rm struct {
	FileName string   `clop:"short;long" usage:"config filename" valid:"required"`
	GateAddr []string `clop:"short;long" usage:""`
}

func (r *Rm) SubMain() {

}
