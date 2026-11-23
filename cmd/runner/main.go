package main

import (
	"fmt"
	"time"

	"github.com/inxups/go-proc-runner/pkg/runner"
)

func main() {
	res := runner.RunWithTimeOut(2*time.Second, "python3", "testdata/hang.py")
	//%v 任意数值，%+v 结构体 %s字符串 %T 类型 %q 带引号和转译符输出 %d 整数
	fmt.Printf("Duration   : %v\n", res.Duration)
	fmt.Printf("TimeOut	   : %v\n", res.TimedOut)
	fmt.Printf("Stdout     : %v\n", res.Stdout)
	fmt.Printf("Error	   : %v\n", res.Stderr)
}
