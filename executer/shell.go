package executer

func init() {
	Register("shell", createShellExecuter)
}
