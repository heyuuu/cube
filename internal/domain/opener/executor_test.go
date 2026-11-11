package opener

// fakeExecutor 记录 Run 调用的 bin+args，不真正启动子进程。
// 注入 Service 供 Open 全链路测试用（Executor 接口的测试替身）。
type fakeExecutor struct {
	calls []fakeCall
}
type fakeCall struct {
	bin  string
	args []string
}

func (f *fakeExecutor) Run(bin string, args ...string) error {
	f.calls = append(f.calls, fakeCall{bin: bin, args: append([]string(nil), args...)})
	return nil
}
