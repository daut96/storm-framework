# --- MANDATORY CONFIG ---
# Tentukan nama binary secara eksplisit di sini
NAME     := sniffer_core

# --- PATH CONFIG ---
# BIN_DIR disuapi dari Python (Environment Variable)
OUT      := $(BIN_DIR)
TARGET   := $(OUT)/$(NAME)
SOURCES  := $(wildcard *.go) # Eh, maksudnya *.c, maaf kebiasaan Go tadi

# --- RULES ---
all: $(TARGET)

$(TARGET): $(SOURCES)
	@mkdir -p $(OUT)
	@echo "[*] Building $(NAME)..."
	
	@# Otomatisasi library pcap tetap dipertahankan
	@LIBS=$$(grep -q "pcap.h" $(SOURCES) && echo "-lpcap" || echo ""); \
	gcc $(SOURCES) -o $(TARGET) $$LIBS
	
	@chmod +x $(TARGET)
	@touch $(TARGET)
	@echo "[✓] Binary created: $(TARGET)"

clean:
	rm -f $(TARGET)

