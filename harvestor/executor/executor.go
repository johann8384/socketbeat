package executor

import "exec"

type ExecPlugin struct {
}

func (executor *ExecPlugin) Run(command string, stderr io.Writer, stdout io.Writer) {
  command := exec.Command("fake.sh")
  command.Stdout = stdout
  command.Stderr = stderr
  command.Start()
  command.Wait()
}
