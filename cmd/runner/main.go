package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/inxups/go-proc-runner/pkg/runner"
)

func main() {
	var tasks []runner.Task
	for i := 1; i <= 6; i++ {
		if i == 3 {
			tasks = append(tasks, runner.Task{
				Timedout: 2 * time.Second,
				Name:     "python3",
				Args:     []string{"testdata/hang.py"},
			})
			continue
		}
		tasks = append(tasks, runner.Task{
			Timedout: 3 * time.Second,
			Name:     "python3",
			Args:     []string{"testdata/work.py", strconv.Itoa(i)},
		})
	}

	start := time.Now()
	results := runner.RunWorkerPool(tasks, 2)
	total := time.Since(start)

	for i, res := range results {
		fmt.Printf("Task: %v | Duration: %v | Timedout %v | Err %v\n",
			i+1, res.Duration.Round(time.Millisecond), res.TimedOut, res.Err)
	}

	fmt.Printf("Total duration: %v\n", total.Round(time.Millisecond))

}
