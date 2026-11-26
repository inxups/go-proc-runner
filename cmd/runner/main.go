package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/inxups/go-proc-runner/pkg/runner"
)

func main() {
	sigCtx, sigCancel := context.WithCancel(context.Background())
	defer sigCancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case sig := <-sigCh:
			fmt.Printf("\nReceived signal: %v, stop dispatching new tasks...\n", sig)
			sigCancel()
		case <-sigCtx.Done():
		}
	}()

	globalCtx, globalCancel := context.WithTimeout(sigCtx, 4*time.Second)
	defer globalCancel()

	tasks := []runner.Task{
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"testdata/work.py", "1"}},
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"testdata/hang.py"}}, // 测单任务超时
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"testdata/work.py", "3"}},
		{Timedout: 10 * time.Second, Name: "python3", Args: []string{"testdata/hang.py"}},     // 测运行中被 SIGTERM 级联强杀
		{Timedout: 10 * time.Second, Name: "python3", Args: []string{"testdata/hang.py"}},     // 同上
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"testdata/work.py", "6"}}, // 测未派发被取消
	}

	options := runner.PoolOptions{
		FailFast:    true,
		WorkerCount: 2,
	}
	start := time.Now()
	results := runner.RunWorkerPool(globalCtx, tasks, options)
	total := time.Since(start)

	for i, res := range results {
		fmt.Printf("Task: %v | Duration: %v | Timedout: %v | Canceled: %v | Err: %v\n",
			i+1, res.Duration.Round(time.Millisecond), res.Timedout, res.Canceled, res.Err)
	}

	fmt.Printf("Total duration: %v\n", total.Round(time.Millisecond))

}
