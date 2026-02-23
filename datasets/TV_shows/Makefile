# TV Shows Dataset Pipeline
#
# Usage:
#   make parse SHOW=breaking_bad   Parse one show's SRTs
#   make parse-all                  Parse all shows
#   make unique                     Build unique + common_words
#   make pool                       Build common_pool
#   make all                        Full pipeline (parse-all → unique → pool)
#   make clean                      Remove generated files
#   make stats                      Print dataset statistics

PYTHON := parser/.venv/bin/python3
PARSER := parser

.PHONY: parse parse-all unique pool all clean stats setup

# Parse a single show (requires SHOW=<name>)
parse:
ifndef SHOW
	$(error Usage: make parse SHOW=<show_name>)
endif
	$(PYTHON) $(PARSER)/parse_show.py --show $(SHOW)

# Parse all shows
parse-all:
	$(PYTHON) $(PARSER)/parse_show.py --all

# Build unique words per show + common words
unique:
	$(PYTHON) $(PARSER)/build_unique.py

# Build common pool (words minus excludes)
pool:
	$(PYTHON) $(PARSER)/build_pool.py

# Full pipeline
all: parse-all unique pool

# Remove generated files (keeps exclude CSVs and SRTs)
clean:
	rm -f common_pool.csv common_words.csv common_words.txt
	@for dir in $$($(PYTHON) -c "import sys; sys.path.insert(0,'parser'); from config import load_shows; [print(s) for s in load_shows()]"); do \
		rm -f $$dir/$$dir.csv $$dir/$${dir}_words.txt $$dir/$${dir}_unique.csv; \
	done
	@echo "Cleaned generated files."

# Print statistics
stats:
	@echo "=== Dataset Statistics ==="
	@for dir in $$($(PYTHON) -c "import sys; sys.path.insert(0,'parser'); from config import load_shows; [print(s) for s in load_shows()]"); do \
		words=$$(wc -l < $$dir/$${dir}_words.txt 2>/dev/null || echo "N/A"); \
		unique=$$(tail -n +2 $$dir/$${dir}_unique.csv 2>/dev/null | wc -l || echo "N/A"); \
		exclude=$$(tail -n +2 $$dir/$${dir}_exclude.csv 2>/dev/null | wc -l || echo "N/A"); \
		printf "  %-20s words: %6s  unique: %5s  exclude: %5s\n" "$$dir" "$$words" "$$unique" "$$exclude"; \
	done
	@pool=$$(tail -n +2 common_pool.csv 2>/dev/null | wc -l || echo "N/A"); \
	common=$$(tail -n +2 common_words.csv 2>/dev/null | wc -l || echo "N/A"); \
	echo "  ---"; \
	printf "  %-20s %s\n" "common_pool.csv" "$$pool words"; \
	printf "  %-20s %s\n" "common_words.csv" "$$common words"

# Install dependencies
setup:
	cd $(PARSER) && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt
