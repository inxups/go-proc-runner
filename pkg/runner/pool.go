package runner

import "sync"

type job struct {
	index int
	task  Task
}

type jobResult struct {
	index  int
	result ExecResult
}

func RunWorkerPool(tasks []Task, workerCount int) []ExecResult {
	if workerCount <= 0 {
		workerCount = 1
	}
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}

	jobs := make(chan job, len(tasks))
	results := make(chan jobResult, len(tasks))

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				res := RunWithTimeOut(j.task.Timedout, j.task.Name, j.task.Args...)
				results <- jobResult{
					index:  j.index,
					result: res,
				}
			}
		}()
	}

	for i, j := range tasks {
		jobs <- job{
			index: i,
			task:  j,
		}
	}
	close(jobs)

	wg.Wait()
	close(results)

	output := make([]ExecResult, len(tasks))
	for r := range results {
		output[r.index] = r.result
	}

	return output
}
