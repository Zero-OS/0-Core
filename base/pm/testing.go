package pm

import "syscall"

//TestingPIDTable is used for testing to mock the process manager
type TestingPIDTable struct{}

//RegisterPID notify the process manager that a process has been started
//with the given PID
func (t *TestingPIDTable) RegisterPID(g GetPID) (int, error) {
	return g()
}

//WaitPID waits for a PID until it exits
func (t *TestingPIDTable) WaitPID(pid int) syscall.WaitStatus {
	var status syscall.WaitStatus
	syscall.Wait4(pid, &status, 0, nil)
	return status
}
