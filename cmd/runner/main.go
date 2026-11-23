package main

import (
	"fmt"
	"time"

	"github.com/inxups/go-proc-runner/pkg/runner"
)

func main() {
	res := runner.RunWithTimeOut(2*time.Second, "python3", "testdata/hang.py")
	fmt.Printf("Duration   : %v\n", res.Duration)
	fmt.Printf("TimeOut	   : %v\n", res.TimedOut)
	fmt.Printf("Stdout     : %v\n", res.Stdout)
	fmt.Printf("Error	   : %v\n", res.Stderr)
}
