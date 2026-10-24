.PHONY: test build start stop e2e
test:
	./scripts/go-local.sh test ./...
	python3 -m unittest discover -s workloads -v
build:
	./scripts/go-local.sh build -o .runtime/workbench ./cmd/workbench
start:
	./scripts/start.sh k8s-sim
stop:
	./scripts/stop.sh
e2e:
	python3 scripts/e2e.py
