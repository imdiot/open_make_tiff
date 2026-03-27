.PHONY: build-mac build-windows dev clean \
        vcpkg-install vcpkg-install-arm64 vcpkg-install-x64 vcpkg-install-universal \
        vcpkg-clean vcpkg-rebuild

build-mac:
	wails build -platform darwin/universal

build-windows:
	wails build -platform windows/amd64

dev:
	wails dev

clean:
	rm -rf build/bin/*


# Vcpkg configuration
VCPKG_ROOT ?= $(HOME)/vcpkg
VCPKG = $(VCPKG_ROOT)/vcpkg
OVERLAY_PORTS = third-party/vcpkg/ports
VCPKG_INSTALLED = $(VCPKG_ROOT)/installed
OUTPUT_DIR = third-party/macos-universal

# Vcpkg build targets
# Install all dependencies (universal)
vcpkg-install: vcpkg-install-universal

# ARM64 only
vcpkg-install-arm64:
	$(VCPKG) install \
		libraw[6by9rpi,dng-lossy,dngsdk,rawspeed,x3ftools] \
		tiff[cxx,jpeg,lerc,libdeflate,lzma,tools,webp,zip,zstd] \
		--overlay-ports=$(OVERLAY_PORTS) \
		--triplet=arm64-osx-release

# x64 only
vcpkg-install-x64:
	$(VCPKG) install \
		libraw[6by9rpi,dng-lossy,dngsdk,rawspeed,x3ftools] \
		tiff[cxx,jpeg,lerc,libdeflate,lzma,tools,webp,zip,zstd] \
		--overlay-ports=$(OVERLAY_PORTS) \
		--triplet=x64-osx-release

# Merge universal binaries
vcpkg-install-universal: vcpkg-install-arm64 vcpkg-install-x64
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

# Clean vcpkg build artifacts
vcpkg-clean:
	$(VCPKG) remove libraw tiff --triplet=arm64-osx-release-release
	$(VCPKG) remove libraw tiff --triplet=x64-osx-release
	rm -f $(OUTPUT_DIR)/dcraw_emu $(OUTPUT_DIR)/tiffcp

# Rebuild
vcpkg-rebuild: vcpkg-clean vcpkg-install
