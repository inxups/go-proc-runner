package runner

import (
	"sync"
	"time"
)

type Task struct {
	Timedout time.Duration
	Name     string
	Args     []string
}

func RunBatch(tasks []Task, maxconcurrency int) []ExecResult {
	if maxconcurrency <= 0 {
		maxconcurrency = 1
	}

	results := make([]ExecResult, len(tasks))
	sem := make(chan struct{}, maxconcurrency)
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Task) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = RunWithTimeOut(t.Timedout, t.Name, t.Args...)
		}(i, task)
	}

	wg.Wait()
	return results
}
