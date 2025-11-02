UNAME := asrClient
TARGET := asrClient
BUILD_DIR := build
WAV_DIR := ./sampleWav/


build:
	@echo "Building for $(UNAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(UNAME) .
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)/*
	@echo "Clean completed"
run:
	@echo "Building for $(UNAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(UNAME) .
	@echo "Running application..."
	@$(BUILD_DIR)/$(UNAME) $(WAV_DIR)
