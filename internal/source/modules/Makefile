# --- Configuration & Failsafe ---
OUT_DIR ?= build
OUT     := $(OUT_DIR)/modules

# --- Target Discovery (Recursive) ---
ALL_SRCS  := $(shell find . -type f -name "*.go")
MAIN_SRCS := $(shell grep -l -E "^package main" $(ALL_SRCS) 2>/dev/null)

# --- Path Mapping & Transformation ---
TARGETS := $(patsubst ./%.go,$(OUT)/%,$(MAIN_SRCS))

# --- Rules ---
.PHONY: all clean pre-build

all: pre-build $(TARGETS)

# Anticipation
pre-build:
	@mkdir -p $(OUT)

# Pattern Rule for dynamic binary compilation
$(OUT)/%: ./%.go
	@mkdir -p $(dir $@)
	@go build -o $@ $<

clean:
	rm -rf $(OUT)
