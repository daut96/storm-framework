# --- Configuration ---
OUT      := $(OUT_DIR)/modules
ifeq ($(strip $(OUT)),)
endif

# --- Target Discovery (Recursive) ---
ALL_SRCS := $(shell find . -type f -name "*.go")

# Filter only files that contain the main package
MAIN_SRCS := $(shell grep -l -E "^package main" $(ALL_SRCS) 2>/dev/null)

# --- Path Mapping & Transformation ---
TARGETS := $(patsubst ./%.go,$(OUT)/%,$(basename $(MAIN_SRCS)))

# --- Rules ---
.PHONY: all clean

# Default target: build all found binaries
all: $(TARGETS)

# Pattern Rule for dynamic binary compilation
# % capture sub-directory structure and file names
$(OUT)/%: ./%.go
	@mkdir -p $(dir $@)
	@go build -o $@ $<

clean:
	rm -rf $(OUT)
