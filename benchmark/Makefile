PYTHON     	?= python3

.PHONY: bench summary clean

bench: 
	@mkdir -p results
	go test -bench=. -benchmem -count=2 -timeout=5m | tee bench_raw.txt

summary: bench
	$(PYTHON) gen_summary.py
	@echo ""
	@echo "Summary in README.md"

protobuf:
	protoc --go_out=. --go_opt=paths=source_relative proto/player.proto

clean:
	rm -rf results
