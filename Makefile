# --- Konfigurasi Centralized Rust ---
ROOT_DIR := $(shell pwd)
SHARED_TARGET := $(ROOT_DIR)/lib/smf/core/cache/rust-session
export CARGO_TARGET_DIR := $(SHARED_TARGET)

# --- Pencarian Module ---
# Cari folder yang punya Makefile (untuk C/Go)
C_GO_DIRS := $(dir $(wildcard external/source/**/Makefile))
# Cari folder yang punya Cargo.toml (untuk Rust)
RUST_PROJECTS := $(dir $(wildcard external/source/**/Cargo.toml))

.PHONY: all clean rust_modules c_go_modules

all: c_go_modules rust_modules
	@echo "[✓] Build Framework Selesai (Offline Mode Active)"

# --- Eksekusi Rust (Terpusat & Offline) ---
rust_modules:
	@for dir in $(RUST_PROJECTS); do \
		echo "[*] Building Rust (Offline) => $$dir"; \
		cd $(ROOT_DIR)/$$dir && cargo build --release --offline; \
	done

# --- Eksekusi C/Go ---
c_go_modules:
	@for dir in $(C_GO_DIRS); do \
		$(MAKE) -C $$dir; \
	done

clean:
	rm -rf $(SHARED_TARGET)
	@for dir in $(C_GO_DIRS); do $(MAKE) -C $$dir clean; done
