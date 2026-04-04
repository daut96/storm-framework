NAME     := smtp_flood
OUT      := $(BIN_DIR)
TARGET   := $(OUT)/$(NAME)

all: $(TARGET)

$(TARGET): *.go
	@mkdir -p $(OUT)
	@go build -o $(TARGET) .
	@chmod +x $(TARGET)
	@touch $(TARGET)

