NAME     := ftp_flood
OUT      := $(BIN_DIR)
TARGET   := $(OUT)/$(NAME)

all: $(TARGET)

$(TARGET): *.go
	@mkdir -p $(OUT)
	@echo "[*] Building Go Module: $(NAME)..."
	@# Compile langsung menggunakan context dari root
	go build -o $(TARGET) .
	@chmod +x $(TARGET)
	@touch $(TARGET)
