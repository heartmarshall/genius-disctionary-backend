# Lyrics Dataset Pipeline
#
# Usage:
#   make pipeline GENIUS_TOKEN=...  Full pipeline (download + dataset + vocabulary)
#   make pipeline-skip-download     Rebuild without re-downloading
#   make pipeline-only ONLY="Adele" Process specific artists
#   make vocabulary                  Rebuild dataset.csv + words.txt for all artists
#   make unique                      Build unique + common_words
#   make pool                        Build common_pool
#   make all                         Full rebuild (vocabulary → unique → pool)
#   make clean                       Remove generated files
#   make stats                       Print dataset statistics

PYTHON := pipeline/.venv/bin/python3
PIPELINE := pipeline

.PHONY: pipeline pipeline-skip-download pipeline-only vocabulary unique pool all clean stats setup

# Full pipeline: download + build dataset + build vocabulary
# Usage: make pipeline GENIUS_TOKEN=your-token-here
pipeline:
	cd $(PIPELINE) && GENIUS_TOKEN=$(GENIUS_TOKEN) python3 pipeline.py

# Rebuild dataset + vocabulary without re-downloading
pipeline-skip-download:
	cd $(PIPELINE) && python3 pipeline.py --skip-download

# Process specific artists only (skip download)
# Usage: make pipeline-only ONLY="Adele,Queen"
pipeline-only:
	cd $(PIPELINE) && python3 pipeline.py --skip-download --only "$(ONLY)"

# Build vocabulary (dataset.csv + words.txt) for all artists
vocabulary:
	$(PYTHON) $(PIPELINE)/build_vocabulary.py

# Build unique words per artist + common words
unique:
	$(PYTHON) $(PIPELINE)/build_unique.py

# Build common pool (words minus excludes)
pool:
	$(PYTHON) $(PIPELINE)/build_pool.py

# Full rebuild: vocabulary → unique → pool
all: vocabulary unique pool

# Remove generated files (keeps exclude CSVs and sources)
clean:
	rm -f output/common_pool.csv output/common_words.csv output/common_words.txt
	@for dir in output/*/; do \
		rm -f "$$dir/dataset.csv" "$$dir/words.txt" "$$dir/unique.csv"; \
	done
	@echo "Cleaned generated files."

# Print statistics
stats:
	@echo "=== Dataset Statistics ==="
	@for dir in output/*/; do \
		name=$$(basename "$$dir"); \
		words=$$(wc -l < "$$dir/words.txt" 2>/dev/null || echo "N/A"); \
		unique=$$(tail -n +2 "$$dir/unique.csv" 2>/dev/null | wc -l || echo "N/A"); \
		exclude=$$(tail -n +2 "$$dir/exclude.csv" 2>/dev/null | wc -l || echo "N/A"); \
		printf "  %-20s words: %6s  unique: %5s  exclude: %5s\n" "$$name" "$$words" "$$unique" "$$exclude"; \
	done
	@pool=$$(tail -n +2 output/common_pool.csv 2>/dev/null | wc -l || echo "N/A"); \
	common=$$(tail -n +2 output/common_words.csv 2>/dev/null | wc -l || echo "N/A"); \
	echo "  ---"; \
	printf "  %-20s %s\n" "common_pool.csv" "$$pool words"; \
	printf "  %-20s %s\n" "common_words.csv" "$$common words"

# Install dependencies
setup:
	cd $(PIPELINE) && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt && .venv/bin/python3 -m spacy download en_core_web_sm
