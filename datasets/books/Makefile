# Books Dataset Pipeline
#
# Usage:
#   make parse BOOK=animal_farm    Parse one book's EPUB
#   make parse-all                  Parse all books
#   make unique                     Build unique + common_words
#   make pool                       Build common_pool
#   make all                        Full pipeline (parse-all → unique → pool)
#   make clean                      Remove generated files
#   make stats                      Print dataset statistics

PYTHON := parser/.venv/bin/python3
PARSER := parser

.PHONY: parse parse-all unique pool all clean stats setup

# Parse a single book (requires BOOK=<name>)
parse:
ifndef BOOK
	$(error Usage: make parse BOOK=<book_name>)
endif
	$(PYTHON) $(PARSER)/parse_book.py --book $(BOOK)

# Parse all books
parse-all:
	$(PYTHON) $(PARSER)/parse_book.py --all

# Build unique words per book + common words
unique:
	$(PYTHON) $(PARSER)/build_unique.py

# Build common pool (words minus excludes)
pool:
	$(PYTHON) $(PARSER)/build_pool.py

# Full pipeline
all: parse-all unique pool

# Remove generated files (keeps exclude CSVs and EPUBs)
clean:
	rm -f common_pool.csv common_words.csv common_words.txt
	@for dir in $$($(PYTHON) -c "import sys; sys.path.insert(0,'parser'); from config import load_books; [print(b) for b in load_books()]"); do \
		rm -f $$dir/$$dir.csv $$dir/$${dir}_words.txt $$dir/$${dir}_unique.csv; \
	done
	@echo "Cleaned generated files."

# Print statistics
stats:
	@echo "=== Dataset Statistics ==="
	@for dir in $$($(PYTHON) -c "import sys; sys.path.insert(0,'parser'); from config import load_books; [print(b) for b in load_books()]"); do \
		words=$$(wc -l < $$dir/$${dir}_words.txt 2>/dev/null || echo "N/A"); \
		unique=$$(tail -n +2 $$dir/$${dir}_unique.csv 2>/dev/null | wc -l || echo "N/A"); \
		exclude=$$(tail -n +2 $$dir/$${dir}_exclude.csv 2>/dev/null | wc -l || echo "N/A"); \
		printf "  %-30s words: %6s  unique: %5s  exclude: %5s\n" "$$dir" "$$words" "$$unique" "$$exclude"; \
	done
	@pool=$$(tail -n +2 common_pool.csv 2>/dev/null | wc -l || echo "N/A"); \
	common=$$(tail -n +2 common_words.csv 2>/dev/null | wc -l || echo "N/A"); \
	echo "  ---"; \
	printf "  %-30s %s\n" "common_pool.csv" "$$pool words"; \
	printf "  %-30s %s\n" "common_words.csv" "$$common words"

# Install dependencies
setup:
	cd $(PARSER) && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt && .venv/bin/python3 -m spacy download en_core_web_sm
