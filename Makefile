.PHONY: build-mac build-windows dev clean \
        vcpkg-install vcpkg-install-macos-arm64 vcpkg-install-macos-x64 vcpkg-install-macos-universal \
        vcpkg-install-windows vcpkg-clean-windows vcpkg-rebuild-windows \
        vcpkg-clean-macos vcpkg-rebuild-macos \
        vcpkg-clean vcpkg-rebuild

# Vcpkg configuration (must be before PKG_CONFIG_PATH)
ifeq ($(OS),Windows_NT)
VCPKG_ROOT ?= $(USERPROFILE)/vcpkg
else
VCPKG_ROOT ?= $(HOME)/vcpkg
endif

# Detect platform-specific triplet
ifeq ($(OS),Windows_NT)
TRIPLET := x64-windows-static-release
else
TRIPLET := universal-osx-release
endif
PKG_CONFIG_PATH = $(VCPKG_ROOT)/installed/$(TRIPLET)/lib/pkgconfig

build-mac: export PKG_CONFIG_PATH := $(VCPKG_ROOT)/installed/universal-osx-release/lib/pkgconfig
build-mac:
	wails build -platform darwin/universal

build-windows: export PKG_CONFIG_PATH := $(VCPKG_ROOT)/installed/x64-windows-static-release/lib/pkgconfig
build-windows:
	wails build -platform windows/amd64

dev: export PKG_CONFIG_PATH := $(PKG_CONFIG_PATH)
dev:
	wails dev

clean:
	rm -rf build/bin/*


# Vcpkg tools
VCPKG = $(VCPKG_ROOT)/vcpkg
ifeq ($(OS),Windows_NT)
VCPKG = $(VCPKG_ROOT)/vcpkg.exe
endif
OVERLAY_PORTS = third-party/vcpkg/ports
OVERLAY_TRIPLETS = third-party/vcpkg/triplets
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
		--overlay-triplets=$(OVERLAY_TRIPLETS) \
		--triplet=arm64-osx-release

# macOS x64 only
vcpkg-install-macos-x64:
	$(VCPKG) install \
		libraw[6by9rpi,dng-lossy,dngsdk,rawspeed,x3ftools] \
		tiff[cxx,jpeg,lerc,libdeflate,lzma,tools,webp,zip,zstd] \
		--overlay-ports=$(OVERLAY_PORTS) \
		--overlay-triplets=$(OVERLAY_TRIPLETS) \
		--triplet=x64-osx-release

# Merge arm64 + x64 into universal fat libraries
vcpkg-install-macos-universal: vcpkg-install-macos-arm64 vcpkg-install-macos-x64
	@echo "Merging static libraries into universal-osx-release..."
	@rm -rf $(VCPKG_INSTALLED)/universal-osx-release
	@mkdir -p $(VCPKG_INSTALLED)/universal-osx-release/lib/pkgconfig
	@cp -R $(VCPKG_INSTALLED)/arm64-osx-release/include $(VCPKG_INSTALLED)/universal-osx-release/
	@for f in $(VCPKG_INSTALLED)/arm64-osx-release/lib/*.a; do \
		lib=$$(basename $$f); \
		if [ -f "$(VCPKG_INSTALLED)/x64-osx-release/lib/$$lib" ]; then \
			lipo -create $$f $(VCPKG_INSTALLED)/x64-osx-release/lib/$$lib \
				-output $(VCPKG_INSTALLED)/universal-osx-release/lib/$$lib; \
		else \
			cp $$f $(VCPKG_INSTALLED)/universal-osx-release/lib/$$lib; \
		fi; \
	done
	@cp $(VCPKG_INSTALLED)/arm64-osx-release/lib/pkgconfig/*.pc \
		$(VCPKG_INSTALLED)/universal-osx-release/lib/pkgconfig/
	@if [ -f "$(VCPKG_INSTALLED)/arm64-osx-release/tools/tiff/tiffcp" ] && \
	    [ -f "$(VCPKG_INSTALLED)/x64-osx-release/tools/tiff/tiffcp" ]; then \
		mkdir -p $(OUTPUT_DIR); \
		lipo -create \
			$(VCPKG_INSTALLED)/arm64-osx-release/tools/tiff/tiffcp \
			$(VCPKG_INSTALLED)/x64-osx-release/tools/tiff/tiffcp \
			-output $(OUTPUT_DIR)/tiffcp; \
		chmod +x $(OUTPUT_DIR)/tiffcp; \
		echo "Universal tiffcp created at: $(OUTPUT_DIR)/tiffcp"; \
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
	rm -rf $(VCPKG_INSTALLED)/universal-osx-release
	rm -f $(OUTPUT_DIR)/tiffcp

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
	@powershell -Command "Copy-Item '$(VCPKG_INSTALLED)/x64-windows-static-release/tools/tiff/tiffcp.exe' -Destination 'third-party/windows-x64/' -ErrorAction SilentlyContinue"
	@echo "Static binaries copied (no DLL dependencies)"

vcpkg-clean-windows:
	$(VCPKG) remove libraw tiff --triplet=x64-windows-static-release --recurse
	@powershell -Command "Remove-Item 'third-party/windows-x64/tiffcp.exe' -ErrorAction SilentlyContinue"

vcpkg-rebuild-windows: vcpkg-clean-windows vcpkg-install-windows
