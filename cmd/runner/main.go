package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/inxups/go-proc-runner/pkg/runner"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case sig := <-sigCh:
			fmt.Printf("\nReceived signal: %v, stop dispatching new tasks...\n", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	var tasks []runner.Task
	for i := 1; i <= 8; i++ {
		tasks = append(tasks, runner.Task{
			Timedout: 3 * time.Second,
			Name:     "python3",
			Args:     []string{"testdata/work.py", strconv.Itoa(i)},
		})
	}

	fmt.Printf("Starting worker pool (pid: %v), plz Ctrl+C to stop... ", os.Getpid())
	start := time.Now()
	results := runner.RunWorkerPool(ctx, tasks, 2)
	total := time.Since(start)

	for i, res := range results {
		fmt.Printf("Task: %v | Duration: %v | Timedout %v | Err %v\n",
			i+1, res.Duration.Round(time.Millisecond), res.TimedOut, res.Err)
	}

	fmt.Printf("Total duration: %v\n", total.Round(time.Millisecond))

}
