# --- 1. Konfigurasi Global (Pusat Kendali) ---
ROOT_DIR := $(shell pwd)
export BIN_DIR := $(ROOT_DIR)/external/source/bin
export SHARED_TARGET := $(ROOT_DIR)/lib/smf/core/cache/rust-session
export CARGO_TARGET_DIR := $(SHARED_TARGET)

# --- 2. Pencarian Otomatis (Deep Scan) ---
# Mencari folder yang ada Makefile (C/Assembly/Custom)
C_DIRS := $(shell find . -name "Makefile" -not -path "./Makefile" -not -path "*/.*" | xargs -I {} dirname {})

# Mencari folder yang ada file .go
GO_DIRS := $(shell find . -name "*.go" -not -path "*/.*" | xargs -I {} dirname {} | sort | uniq)

# Mencari folder yang ada Cargo.toml (Rust)
RUST_DIRS := $(shell find . -name "Cargo.toml" -not -path "*/target/*" -not -path "*/.*" | xargs -I {} dirname {})

.PHONY: all prepare build_c build_go build_rust clean

# --- 3. Target Utama (Urutan Kerja) ---
all: prepare build_c build_go build_rust

prepare:
	@mkdir -p $(BIN_DIR)
	@mkdir -p $(SHARED_TARGET)

# --- 4. Logika Build C / Makefile Lainnya ---
build_c:
	@for dir in $(C_DIRS); do \
		echo "[*] Triggering Makefile: $$dir"; \
		$(MAKE) -C $(ROOT_DIR)/$$dir; \
	done

# --- 5. Logika Build Go (Init + Tidy + Build) ---
build_go:
	@for dir in $(GO_DIRS); do \
		echo "[*] Processing Go Module: $$dir"; \
		cd $(ROOT_DIR)/$$dir; \
		if [ ! -f go.mod ]; then \
			go mod init github.com/storm/$(notdir $$dir); \
		fi; \
		go mod tidy; \
		go build -o $(BIN_DIR)/$(notdir $$dir) .; \
	done

# --- 6. Logika Build Rust (Offline & Centralized) ---
build_rust:
	@for dir in $(RUST_DIRS); do \
		echo "[*] Building Rust (Offline): $$dir"; \
		cd $(ROOT_DIR)/$$dir && cargo build --release --offline; \
		# Opsional: pindahkan hasil binari ke BIN_DIR \
		cp $(SHARED_TARGET)/release/$(notdir $$dir) $(BIN_DIR)/ 2>/dev/null || true; \
	done

clean:
	rm -rf $(BIN_DIR)/*
	rm -rf $(SHARED_TARGET)
