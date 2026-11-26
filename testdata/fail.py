import time
import sys

print("Task falling soon...", flush=True)
time.sleep(0.5)
print("Something went wrong!", file=sys.stderr, flush=True)
sys.exit(0)