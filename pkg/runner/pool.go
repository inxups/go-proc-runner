package runner

import (
	"context"
	"errors"
	"sync"
)

type job struct {
	index int
	task  Task
}

type jobResult struct {
	index  int
	result ExecResult
}

func RunWorkerPool(ctx context.Context, tasks []Task, workerCount int) []ExecResult {
	if len(tasks) == 0 {
		return nil
	}
	if workerCount <= 0 {
		workerCount = 1
	}
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}

	jobs := make(chan job)
	results := make(chan jobResult, len(tasks))

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				res := RunWithContext(ctx, j.task.Timedout, j.task.Name, j.task.Args...)
				results <- jobResult{
					index:  j.index,
					result: res,
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i, j := range tasks {
			if ctx.Err() != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			case jobs <- job{index: i, task: j}:
			}
		}
	}()

	wg.Wait()
	close(results)

	output := make([]ExecResult, len(tasks))
	executed := make([]bool, len(tasks))
	for r := range results {
		output[r.index] = r.result
		executed[r.index] = true
	}

	for o := range output {
		if !executed[o] {
			output[o].Err = ctx.Err()
			output[o].Canceled = errors.Is(ctx.Err(), context.Canceled)
			output[o].TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		}
	}

	return output
}
