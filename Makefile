.PHONY: build build-mac build-windows dev clean \
        vcpkg-install vcpkg-install-macos-arm64 vcpkg-install-macos-x64 vcpkg-install-macos-universal \
        vcpkg-install-windows vcpkg-clean-windows vcpkg-rebuild-windows \
        vcpkg-clean-macos vcpkg-rebuild-macos \
        vcpkg-clean vcpkg-rebuild

# Platform detection (consolidated)
ifeq ($(OS),Windows_NT)
VCPKG_ROOT       ?= $(USERPROFILE)/vcpkg
VCPKG            := $(VCPKG_ROOT)/vcpkg.exe
TRIPLET          := x64-mingw-static-release
_VCPKG_INSTALL   := vcpkg-install-windows
_VCPKG_CLEAN     := vcpkg-clean-windows
_VCPKG_REBUILD   := vcpkg-rebuild-windows
_BUILD_TARGET    := build-windows
else
VCPKG_ROOT       ?= $(HOME)/vcpkg
VCPKG            := $(VCPKG_ROOT)/vcpkg
TRIPLET          := universal-osx-release
_VCPKG_INSTALL   := vcpkg-install-macos-universal
_VCPKG_CLEAN     := vcpkg-clean-macos
_VCPKG_REBUILD   := vcpkg-rebuild-macos
_BUILD_TARGET    := build-mac
endif

# Package configuration
VCPKG_PACKAGES = \
	libraw[6by9rpi,dng-lossy,dngsdk,rawspeed,x3ftools] \
	tiff[cxx,jpeg,lerc,libdeflate,lzma,webp,zip,zstd]

OVERLAY_PORTS    = third-party/vcpkg/ports
OVERLAY_TRIPLETS = third-party/vcpkg/triplets
VCPKG_INSTALLED  = $(VCPKG_ROOT)/installed

# Triplet identifiers
TRIPLET_ARM64     := arm64-osx-release
TRIPLET_X64_MAC   := x64-osx-release
TRIPLET_UNIVERSAL := universal-osx-release
TRIPLET_WINDOWS   := x64-mingw-static-release

# Derived paths
INSTALLED_ARM64     := $(VCPKG_INSTALLED)/$(TRIPLET_ARM64)
INSTALLED_X64_MAC   := $(VCPKG_INSTALLED)/$(TRIPLET_X64_MAC)
INSTALLED_UNIVERSAL := $(VCPKG_INSTALLED)/$(TRIPLET_UNIVERSAL)
INSTALLED_WINDOWS   := $(VCPKG_INSTALLED)/$(TRIPLET_WINDOWS)

PKG_CONFIG_MAC    := $(INSTALLED_UNIVERSAL)/lib/pkgconfig
PKG_CONFIG_WINDOWS := $(INSTALLED_WINDOWS)/lib/pkgconfig

# ── Build targets ────────────────────────────────────────────────

build:
	$(MAKE) $(_BUILD_TARGET)

build-mac: export PKG_CONFIG_PATH := $(PKG_CONFIG_MAC)
build-mac:
	wails build -platform darwin/universal

build-windows: export PKG_CONFIG_PATH := $(PKG_CONFIG_WINDOWS)
build-windows:
	wails build -platform windows/amd64

dev: export PKG_CONFIG_PATH := $(VCPKG_INSTALLED)/$(TRIPLET)/lib/pkgconfig
dev:
	wails dev

clean:
	rm -rf build/bin/*

# ── vcpkg dispatch targets (auto-detect platform) ────────────────

vcpkg-install:
	$(MAKE) $(_VCPKG_INSTALL)

vcpkg-clean:
	$(MAKE) $(_VCPKG_CLEAN)

vcpkg-rebuild:
	$(MAKE) $(_VCPKG_REBUILD)

# ── vcpkg macOS targets ──────────────────────────────────────────

vcpkg-install-macos-arm64:
	$(VCPKG) install $(VCPKG_PACKAGES) \
		--overlay-ports=$(OVERLAY_PORTS) \
		--overlay-triplets=$(OVERLAY_TRIPLETS) \
		--triplet=$(TRIPLET_ARM64)

vcpkg-install-macos-x64:
	$(VCPKG) install $(VCPKG_PACKAGES) \
		--overlay-ports=$(OVERLAY_PORTS) \
		--overlay-triplets=$(OVERLAY_TRIPLETS) \
		--triplet=$(TRIPLET_X64_MAC)

# Merge arm64 + x64 into universal fat libraries
vcpkg-install-macos-universal: vcpkg-install-macos-arm64 vcpkg-install-macos-x64
	@echo "Merging static libraries into $(TRIPLET_UNIVERSAL)..."
	@rm -rf $(INSTALLED_UNIVERSAL)
	@mkdir -p $(INSTALLED_UNIVERSAL)/lib/pkgconfig
	@cp -R $(INSTALLED_ARM64)/include $(INSTALLED_UNIVERSAL)/
	@for f in $(INSTALLED_ARM64)/lib/*.a; do \
		lib=$$(basename $$f); \
		if [ -f "$(INSTALLED_X64_MAC)/lib/$$lib" ]; then \
			lipo -create $$f $(INSTALLED_X64_MAC)/lib/$$lib \
				-output $(INSTALLED_UNIVERSAL)/lib/$$lib; \
		else \
			cp $$f $(INSTALLED_UNIVERSAL)/lib/$$lib; \
		fi; \
	done
	@cp $(INSTALLED_ARM64)/lib/pkgconfig/*.pc \
		$(INSTALLED_UNIVERSAL)/lib/pkgconfig/

vcpkg-clean-macos:
	$(VCPKG) remove libraw tiff --triplet=$(TRIPLET_ARM64)
	$(VCPKG) remove libraw tiff --triplet=$(TRIPLET_X64_MAC)
	rm -rf $(INSTALLED_UNIVERSAL)

vcpkg-rebuild-macos: vcpkg-clean-macos vcpkg-install-macos-universal

# ── vcpkg Windows targets ────────────────────────────────────────

# --overlay-triplets not needed: x64-mingw-static-release is a built-in vcpkg triplet
vcpkg-install-windows:
	$(VCPKG) install $(VCPKG_PACKAGES) \
		--overlay-ports=$(OVERLAY_PORTS) \
		--triplet=$(TRIPLET_WINDOWS) \
		--recurse

vcpkg-clean-windows:
	$(VCPKG) remove libraw tiff adobe-dng-sdk --triplet=$(TRIPLET_WINDOWS) --recurse

vcpkg-rebuild-windows: vcpkg-clean-windows vcpkg-install-windows
