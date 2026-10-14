.PHONY: demo test check report
demo:            ## 1 spec end to end (falls back to a labeled mock model if the API is down)
	python3 -m specgen demo specs/dev/gpu-job-watcher.yaml --model $${MODEL:-nemotron-ultra}
test:
	python3 -m pytest tests -q
check:
	python3 scripts/check_env.py
report:
	python3 -m specgen report --run-id dev-3 --run-id heldout-1
