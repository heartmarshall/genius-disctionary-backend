.PHONY: pipeline pipeline-skip-download pipeline-only vocabulary common-words

# Full pipeline: download + build dataset + build vocabulary
# Usage: make pipeline GENIUS_TOKEN=your-token-here
pipeline:
	cd pipeline && GENIUS_TOKEN=$(GENIUS_TOKEN) python3 pipeline.py

# Rebuild dataset + vocabulary without re-downloading
pipeline-skip-download:
	cd pipeline && python3 pipeline.py --skip-download

# Process specific artists only (skip download)
# Usage: make pipeline-only ONLY="Adele,Queen"
pipeline-only:
	cd pipeline && python3 pipeline.py --skip-download --only "$(ONLY)"

# Build vocabulary for all artists
vocabulary:
	cd pipeline && python3 build_vocabulary.py

# Build cross-artist common words analysis
common-words:
	cd pipeline && python3 build_common_words.py
