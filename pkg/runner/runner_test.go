package runner

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 1. 表驱动测试：一次性覆盖单任务正常退出、报错退出、自身超时、父级取消四种路径
func TestRunWithContext(t *testing.T) {
	tests := []struct {
		name         string
		script       string
		timeout      time.Duration
		cancelAfter  time.Duration
		wantErr      bool
		wantTimedout bool
		wantCanceled bool
		wantStdout   string
		wantStderr   string
	}{
		{
			name:       "normal exit",
			script:     "print('hello runner')",
			timeout:    3 * time.Second,
			wantStdout: "hello runner",
		},
		{
			name:       "non-zero exit",
			script:     "import sys; print('boom', file=sys.stderr); sys.exit(1)",
			timeout:    3 * time.Second,
			wantErr:    true,
			wantStderr: "boom",
		},
		{
			name:         "single task timeout",
			script:       "import time; print('boot', flush=True); time.sleep(5)",
			timeout:      300 * time.Millisecond,
			wantErr:      true,
			wantTimedout: true,
			wantStdout:   "boot",
		},
		{
			name:         "parent context canceled",
			script:       "import time; time.sleep(5)",
			timeout:      5 * time.Second,
			cancelAfter:  250 * time.Millisecond,
			wantErr:      true,
			wantCanceled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.cancelAfter > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				timer := time.AfterFunc(tc.cancelAfter, cancel)
				defer timer.Stop()
			}

			res := RunWithContext(ctx, tc.timeout, "python3", "-c", tc.script)

			if (res.Err != nil) != tc.wantErr {
				t.Fatalf("Err = %v, wantErr = %v", res.Err, tc.wantErr)
			}
			if res.Timedout != tc.wantTimedout {
				t.Errorf("Timedout = %v, want %v", res.Timedout, tc.wantTimedout)
			}
			if res.Canceled != tc.wantCanceled {
				t.Errorf("Canceled = %v, want %v", res.Canceled, tc.wantCanceled)
			}
			if tc.wantStdout != "" && !strings.Contains(res.Stdout, tc.wantStdout) {
				t.Errorf("Stdout = %q, want to contain %q", res.Stdout, tc.wantStdout)
			}
			if tc.wantStderr != "" && !strings.Contains(res.Stderr, tc.wantStderr) {
				t.Errorf("Stderr = %q, want to contain %q", res.Stderr, tc.wantStderr)
			}
		})
	}
}

// 2. 验证 FailFast：单任务失败后立即强杀运行中的兄弟任务，且未派发任务标记为 Canceled
func TestRunWorkerPool_FailFast(t *testing.T) {
	tasks := []Task{
		{Timedout: 5 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(5)"}},
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"-c", "import sys, time; time.sleep(0.15); sys.exit(1)"}},
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(1)"}},
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(1)"}},
	}

	start := time.Now()
	results := RunWorkerPool(context.Background(), tasks, PoolOptions{
		WorkerCount: 2,
		FailFast:    true,
	})
	elapsed := time.Since(start)

	if len(results) != len(tasks) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(tasks))
	}
	if elapsed > 2*time.Second {
		t.Errorf("FailFast took too long: %v", elapsed)
	}

	// Task 0：运行中被 Task 1 触发的 FailFast 级联取消
	if !results[0].Canceled || results[0].Timedout || results[0].Err == nil {
		t.Errorf("Task 0 unexpected result: %+v", results[0])
	}

	// Task 1：自身报错退出，不是超时也不是被取消
	if results[1].Err == nil || results[1].Timedout || results[1].Canceled {
		t.Errorf("Task 1 unexpected result: %+v", results[1])
	}

	// Task 2~3：未派发（或取消后立即中止），必须标记为 Canceled 且 Err 为 context.Canceled
	for i := 2; i < len(results); i++ {
		if !results[i].Canceled || results[i].Timedout || !errors.Is(results[i].Err, context.Canceled) {
			t.Errorf("Task %d unexpected result: %+v", i, results[i])
		}
	}
}

// 3. 验证 FailFast=false + 全局超时：单任务失败不连坐，到达全局超时后统一截断
func TestRunWorkerPool_GlobalTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	tasks := []Task{
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"-c", "import sys; sys.exit(1)"}},
		{Timedout: 5 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(5)"}},
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"-c", "print('ok')"}},
		{Timedout: 5 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(5)"}},
		{Timedout: 3 * time.Second, Name: "python3", Args: []string{"-c", "print('never')"}},
	}

	results := RunWorkerPool(ctx, tasks, PoolOptions{
		WorkerCount: 2,
		FailFast:    false,
	})

	// Task 0：自身出错，但不会影响后面任务继续执行
	if results[0].Err == nil || results[0].Timedout || results[0].Canceled {
		t.Errorf("Task 0 unexpected result: %+v", results[0])
	}

	// Task 2：在 Task 0 失败后被空闲 worker 领走并顺利跑完
	if results[2].Err != nil || !strings.Contains(results[2].Stdout, "ok") {
		t.Errorf("Task 2 unexpected result: %+v", results[2])
	}

	// Task 1 和 Task 3：在 600ms 全局超时时被截断
	for _, idx := range []int{1, 3} {
		if !results[idx].Timedout || results[idx].Canceled || results[idx].Err == nil {
			t.Errorf("Task %d unexpected result: %+v", idx, results[idx])
		}
	}

	// Task 4：未派发，应按全局超时原因补齐 DeadlineExceeded
	if !results[4].Timedout || results[4].Canceled || !errors.Is(results[4].Err, context.DeadlineExceeded) {
		t.Errorf("Task 4 unexpected result: %+v", results[4])
	}
}

// 4. 边界条件：空任务列表与非法 WorkerCount
func TestRunWorkerPool_EdgeCases(t *testing.T) {
	if res := RunWorkerPool(context.Background(), nil, PoolOptions{WorkerCount: 2}); res != nil {
		t.Errorf("empty tasks should return nil, got %v", res)
	}

	tasks := []Task{
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "print('a')"}},
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "print('b')"}},
	}
	results := RunWorkerPool(context.Background(), tasks, PoolOptions{WorkerCount: 0})
	if len(results) != 2 || results[0].Err != nil || results[1].Err != nil {
		t.Errorf("WorkerCount=0 fallback failed: %+v", results)
	}
}

// 5. 协程泄漏检测：验证各种退出路径下，后台派发协程和 worker 协程全部干净回收
func TestRunWorkerPool_NoGoroutineLeak(t *testing.T) {
	// 先做一次简单预热，稳定初始协程基线
	_ = RunWorkerPool(context.Background(), []Task{
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "pass"}},
	}, PoolOptions{WorkerCount: 1})

	time.Sleep(50 * time.Millisecond)
	before := runtime.NumGoroutine()

	// 场景 A：正常全部跑完
	RunWorkerPool(context.Background(), []Task{
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "pass"}},
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "pass"}},
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "pass"}},
	}, PoolOptions{WorkerCount: 2})

	// 场景 B：外部主动取消
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(150*time.Millisecond, cancel)
	defer timer.Stop()
	defer cancel()

	RunWorkerPool(ctx, []Task{
		{Timedout: 5 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(5)"}},
		{Timedout: 5 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(5)"}},
		{Timedout: 5 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(5)"}},
	}, PoolOptions{WorkerCount: 2})

	// 场景 C：FailFast 提前中止
	RunWorkerPool(context.Background(), []Task{
		{Timedout: 5 * time.Second, Name: "python3", Args: []string{"-c", "import time; time.sleep(5)"}},
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "import sys; sys.exit(1)"}},
		{Timedout: 2 * time.Second, Name: "python3", Args: []string{"-c", "pass"}},
	}, PoolOptions{WorkerCount: 2, FailFast: true})

	// 给调度器留一个极短的收尾窗口，等刚关闭通道的派发协程完全退出
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		after := runtime.NumGoroutine()
		if after <= before {
			t.Logf("goroutines ok: before=%d, after=%d", before, after)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("goroutine leak detected: before=%d, after=%d", before, runtime.NumGoroutine())
}
