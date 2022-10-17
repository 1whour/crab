package start

type Start struct {
	FileName string   `clop:"short;long" usage:"config filename" valid:"required"`
	GateAddr []string `clop:"short;long" usage:""`
}

// 入口函数
func (r *Start) SubMain() {

}
