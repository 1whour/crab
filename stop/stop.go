package stop

type Stop struct {
	FileName string   `clop:"short;long" usage:"config filename" valid:"required"`
	GateAddr []string `clop:"short;long" usage:""`
}

func (s *Stop) SubMain() {

}
