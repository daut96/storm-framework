.SILENT:

RAW_NAME := stls
NAME     := libstls.so
OUT_DIR  ?= build
OUT      := $(OUT_DIR)/core/tls
TARGET   := $(OUT)/$(NAME)

# Cargo cache path resolution
CACHE    := $(CARGO_TARGET_DIR)
ifeq ($(CACHE),)
    CACHE := target
endif

# --- HARDWARE OPTIMIZATION LAYER ---
# Using the Compile-on-Device paradigm. 
# Delegates 100% of microarchitectural resolution to the host LLVM backend.
# rustc will evaluate CPUID (x86_64) or MIDR_EL1 (ARM64) dynamically 
# to inject the most advanced vector instruction sets (AVX, SVE, NEON) 
# supported by the user's local silicon.
export RUSTFLAGS := -C target-cpu=native

# Source code resolution
# Tambahkan build.rs dan folder external ke dalam pemantauan SRC
# agar jika Google submodule di-update, Makefile tahu harus me-rebuild.
SRC := $(shell find src -name "*.rs" 2>/dev/null) \
       $(shell find ../../../../external/source/google/vendor/boringssl -name "*" -type f 2>/dev/null | head -n 20) \
       Cargo.toml build.rs

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(SRC)
	mkdir -p $(OUT)
	cargo build --release --offline --quiet
	cp $(CACHE)/release/lib$(RAW_NAME).so $(TARGET)
	strip --strip-unneeded $(TARGET)

clean:
	cargo clean --quiet
	rm -rf $(OUT)
