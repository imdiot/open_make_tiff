.PHONY: build-mac build-mac-arm64 build-mac-x64 build-windows dev clean \
        vcpkg-install vcpkg-install-macos-arm64 vcpkg-install-macos-x64 vcpkg-install-macos-universal \
        vcpkg-install-windows vcpkg-clean-windows vcpkg-rebuild-windows \
        vcpkg-clean-macos vcpkg-rebuild-macos \
        vcpkg-clean vcpkg-rebuild

# Detect platform-specific vcpkg triplet and pkg-config path
ifeq ($(OS),Windows_NT)
TRIPLET := x64-windows-static-release
else
TRIPLET := $(shell uname -m | sed 's/arm64/arm64-osx-release/' | sed 's/x86_64/x64-osx-release/')
endif
PKG_CONFIG_PATH := $(VCPKG_ROOT)/installed/$(TRIPLET)/lib/pkgconfig

build-mac: export PKG_CONFIG_PATH := $(PKG_CONFIG_PATH)
build-mac:
	wails build -platform darwin/universal

build-mac-arm64: export PKG_CONFIG_PATH := $(VCPKG_ROOT)/installed/arm64-osx-release/lib/pkgconfig
build-mac-arm64:
	wails build -platform darwin/arm64

build-mac-x64: export PKG_CONFIG_PATH := $(VCPKG_ROOT)/installed/x64-osx-release/lib/pkgconfig
build-mac-x64:
	wails build -platform darwin/amd64

build-windows: export PKG_CONFIG_PATH := $(VCPKG_ROOT)/installed/x64-windows-static-release/lib/pkgconfig
build-windows:
	wails build -platform windows/amd64

dev: export PKG_CONFIG_PATH := $(PKG_CONFIG_PATH)
dev:
	wails dev

clean:
	rm -rf build/bin/*


# Vcpkg configuration
# On Windows, prefer USERPROFILE; on Unix, use HOME
ifeq ($(OS),Windows_NT)
VCPKG_ROOT ?= $(USERPROFILE)/vcpkg
else
VCPKG_ROOT ?= $(HOME)/vcpkg
endif

VCPKG = $(VCPKG_ROOT)/vcpkg
ifeq ($(OS),Windows_NT)
VCPKG = $(VCPKG_ROOT)/vcpkg.exe
endif
OVERLAY_PORTS = third-party/vcpkg/ports
VCPKG_INSTALLED = $(VCPKG_ROOT)/installed
OUTPUT_DIR = third-party/macos-universal

# Vcpkg build targets
# Install all dependencies (platform auto-detect)
vcpkg-install:
ifeq ($(OS),Windows_NT)
	$(MAKE) vcpkg-install-windows
else
	$(MAKE) vcpkg-install-macos-universal
endif

# macOS ARM64 only
vcpkg-install-macos-arm64:
	$(VCPKG) install \
		libraw[6by9rpi,dng-lossy,dngsdk,rawspeed,x3ftools] \
		tiff[cxx,jpeg,lerc,libdeflate,lzma,tools,webp,zip,zstd] \
		--overlay-ports=$(OVERLAY_PORTS) \
		--triplet=arm64-osx-release

# macOS x64 only
vcpkg-install-macos-x64:
	$(VCPKG) install \
		libraw[6by9rpi,dng-lossy,dngsdk,rawspeed,x3ftools] \
		tiff[cxx,jpeg,lerc,libdeflate,lzma,tools,webp,zip,zstd] \
		--overlay-ports=$(OVERLAY_PORTS) \
		--triplet=x64-osx-release

# Merge macOS universal binaries
vcpkg-install-macos-universal: vcpkg-install-macos-arm64 vcpkg-install-macos-x64
	@echo "Merging dcraw_emu..."
	@if [ -f "$(VCPKG_INSTALLED)/arm64-osx-release/bin/dcraw_emu" ] && [ -f "$(VCPKG_INSTALLED)/x64-osx-release/bin/dcraw_emu" ]; then \
		lipo -create \
			$(VCPKG_INSTALLED)/arm64-osx-release/bin/dcraw_emu \
			$(VCPKG_INSTALLED)/x64-osx-release/bin/dcraw_emu \
			-output $(OUTPUT_DIR)/dcraw_emu; \
		chmod +x $(OUTPUT_DIR)/dcraw_emu; \
		echo "Universal dcraw_emu created at: $(OUTPUT_DIR)/dcraw_emu"; \
	elif [ -f "$(VCPKG_INSTALLED)/arm64-osx-release/bin/dcraw_emu" ]; then \
		cp $(VCPKG_INSTALLED)/arm64-osx-release/bin/dcraw_emu $(OUTPUT_DIR)/dcraw_emu; \
		chmod +x $(OUTPUT_DIR)/dcraw_emu; \
		echo "dcraw_emu (arm64 only) copied to: $(OUTPUT_DIR)/dcraw_emu"; \
	fi
	@echo "Merging tiffcp..."
	@if [ -f "$(VCPKG_INSTALLED)/arm64-osx-release/tools/tiff/tiffcp" ] && [ -f "$(VCPKG_INSTALLED)/x64-osx-release/tools/tiff/tiffcp" ]; then \
		lipo -create \
			$(VCPKG_INSTALLED)/arm64-osx-release/tools/tiff/tiffcp \
			$(VCPKG_INSTALLED)/x64-osx-release/tools/tiff/tiffcp \
			-output $(OUTPUT_DIR)/tiffcp; \
		chmod +x $(OUTPUT_DIR)/tiffcp; \
		echo "Universal tiffcp created at: $(OUTPUT_DIR)/tiffcp"; \
	elif [ -f "$(VCPKG_INSTALLED)/arm64-osx-release/tools/tiff/tiffcp" ]; then \
		cp $(VCPKG_INSTALLED)/arm64-osx-release/tools/tiff/tiffcp $(OUTPUT_DIR)/tiffcp; \
		chmod +x $(OUTPUT_DIR)/tiffcp; \
		echo "tiffcp (arm64 only) copied to: $(OUTPUT_DIR)/tiffcp"; \
	fi
	@echo "Merging raw-identify..."
	@if [ -f "$(VCPKG_INSTALLED)/arm64-osx-release/bin/raw-identify" ] && [ -f "$(VCPKG_INSTALLED)/x64-osx-release/bin/raw-identify" ]; then \
		lipo -create \
			$(VCPKG_INSTALLED)/arm64-osx-release/bin/raw-identify \
			$(VCPKG_INSTALLED)/x64-osx-release/bin/raw-identify \
			-output $(OUTPUT_DIR)/raw-identify; \
		chmod +x $(OUTPUT_DIR)/raw-identify; \
		echo "Universal raw-identify created at: $(OUTPUT_DIR)/raw-identify"; \
	elif [ -f "$(VCPKG_INSTALLED)/arm64-osx-release/bin/raw-identify" ]; then \
		cp $(VCPKG_INSTALLED)/arm64-osx-release/bin/raw-identify $(OUTPUT_DIR)/raw-identify; \
		chmod +x $(OUTPUT_DIR)/raw-identify; \
		echo "raw-identify (arm64 only) copied to: $(OUTPUT_DIR)/raw-identify"; \
	fi

# Clean vcpkg build artifacts (platform auto-detect)
vcpkg-clean:
ifeq ($(OS),Windows_NT)
	$(MAKE) vcpkg-clean-windows
else
	$(MAKE) vcpkg-clean-macos
endif

# Clean macOS build artifacts
vcpkg-clean-macos:
	$(VCPKG) remove libraw tiff --triplet=arm64-osx-release
	$(VCPKG) remove libraw tiff --triplet=x64-osx-release
	rm -f $(OUTPUT_DIR)/dcraw_emu $(OUTPUT_DIR)/raw-identify $(OUTPUT_DIR)/tiffcp

# Rebuild (platform auto-detect)
vcpkg-rebuild:
ifeq ($(OS),Windows_NT)
	$(MAKE) vcpkg-rebuild-windows
else
	$(MAKE) vcpkg-rebuild-macos
endif

# Rebuild macOS
vcpkg-rebuild-macos: vcpkg-clean-macos vcpkg-install

# Windows vcpkg build targets (static linking)
vcpkg-install-windows:
	$(VCPKG) install \
		libraw[6by9rpi,dng-lossy,dngsdk,rawspeed,x3ftools] \
		tiff[cxx,jpeg,lerc,libdeflate,lzma,tools,webp,zip,zstd] \
		--overlay-ports=$(OVERLAY_PORTS) \
		--triplet=x64-windows-static-release \
		--recurse
	@echo "Copying Windows static binaries..."
	@powershell -Command "Copy-Item '$(VCPKG_INSTALLED)/x64-windows-static-release/bin/dcraw_emu.exe' -Destination 'third-party/windows-x64/' -ErrorAction SilentlyContinue"
	@powershell -Command "Copy-Item '$(VCPKG_INSTALLED)/x64-windows-static-release/bin/raw-identify.exe' -Destination 'third-party/windows-x64/' -ErrorAction SilentlyContinue"
	@powershell -Command "Copy-Item '$(VCPKG_INSTALLED)/x64-windows-static-release/tools/tiff/tiffcp.exe' -Destination 'third-party/windows-x64/' -ErrorAction SilentlyContinue"
	@echo "Static binaries copied (no DLL dependencies)"

vcpkg-clean-windows:
	$(VCPKG) remove libraw tiff --triplet=x64-windows-static-release --recurse
	@powershell -Command "Remove-Item 'third-party/windows-x64/dcraw_emu.exe' -ErrorAction SilentlyContinue"
	@powershell -Command "Remove-Item 'third-party/windows-x64/raw-identify.exe' -ErrorAction SilentlyContinue"
	@powershell -Command "Remove-Item 'third-party/windows-x64/tiffcp.exe' -ErrorAction SilentlyContinue"

vcpkg-rebuild-windows: vcpkg-clean-windows vcpkg-install-windows
