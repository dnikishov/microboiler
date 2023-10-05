.PHONY: report
report:
	gofmt -s -l ./
	go vet ./...
	ineffassign ./...
	gocyclo -over 15 ./
	misspell ./


.PHONY: fix-fmt
fix-fmt:
	gofmt -s -w ./


.PHONY: fix-misspelling
fix-misspelling:
	misspell -w ./


.PHONY: fix
fix: fix-fmt fix-misspelling
