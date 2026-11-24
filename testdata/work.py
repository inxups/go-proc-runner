import sys
import time

task_id = sys.argv[1] if len(sys.argv) > 1 else "0"
print(f"Task {task_id} started.", flush=True)
time.sleep(1)
print(f"Task {task_id} finished.", flush=True)