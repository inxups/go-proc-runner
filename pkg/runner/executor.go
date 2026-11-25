package runner

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// go结构体(把零散，类型不同字段打包使用(访问，func返回值))
// 被打包字段首字母大写可被外部访问(cmd/runner)，小写则只能内部包访问(cmd/runner)
type ExecResult struct {
	Stdout   string
	Stderr   string
	Duration time.Duration
	TimedOut bool
	Canceled bool
	Err      error
}

// ...string 接受多个形参 args...拆包，切片展开
func RunWithContext(patentCtx context.Context, timeout time.Duration, name string, args ...string) ExecResult {
	//:= 创建并使用变量
	ctx, cancel := context.WithTimeout(patentCtx, timeout)
	//defer cancel先别使用，return前再使用，写在这是放忘记和泄漏分支
	//多个defer同stack，先进先出
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	//&指针，cm.Stdout改变，var stdout也改
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	return ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		Canceled: errors.Is(ctx.Err(), context.Canceled),
		TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded),
		Err:      err,
	}
}

func RunWithTimeOut(timedout time.Duration, name string, args ...string) ExecResult {
	return RunWithContext(context.Background(), timedout, name, args...)
}
