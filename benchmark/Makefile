PYTHON     	?= python3

.PHONY: bench summary clean

bench: 
	@mkdir -p results
	go test -bench=. -benchmem -count=2 -timeout=5m | tee bench_raw.txt

summary: bench
	$(PYTHON) gen_summary.py
	@echo ""
	@echo "Summary in README.md"

clean:
	rm -rf results
