package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("C:\\Program Files\\Git\\bin\\bash.exe", "./start-all.sh")
	err := cmd.Run()
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
		fmt.Println(string(output))
	}
}
