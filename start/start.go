package start

type Start struct {
	FileName string   `clop:"short;long" usage:""`
	GateAddr []string `clop:"short;long" usage:""`
}

// 入口函数
func (r *Start) SubMain() {

}
