.PHONY: build build-mac build-windows dev clean \
        vcpkg-install vcpkg-install-macos-arm64 vcpkg-install-macos-x64 vcpkg-install-macos-universal \
        vcpkg-install-windows vcpkg-clean-windows vcpkg-rebuild-windows \
        vcpkg-clean-macos vcpkg-rebuild-macos \
        vcpkg-clean vcpkg-rebuild \
        exiftool-download exiftool-download-windows exiftool-download-macos \
        exiftool-check-windows exiftool-check-macos exiftool-clean

# Platform detection (consolidated)
ifeq ($(OS),Windows_NT)
VCPKG_ROOT       ?= $(USERPROFILE)/vcpkg
VCPKG            := $(VCPKG_ROOT)/vcpkg.exe
TRIPLET          := x64-mingw-static-release
_VCPKG_INSTALL   := vcpkg-install-windows
_VCPKG_CLEAN     := vcpkg-clean-windows
_VCPKG_REBUILD   := vcpkg-rebuild-windows
_BUILD_TARGET    := build-windows
_EXIFTOOL_CHECK  := exiftool-check-windows
else
VCPKG_ROOT       ?= $(HOME)/vcpkg
VCPKG            := $(VCPKG_ROOT)/vcpkg
TRIPLET          := universal-osx-release
_VCPKG_INSTALL   := vcpkg-install-macos-universal
_VCPKG_CLEAN     := vcpkg-clean-macos
_VCPKG_REBUILD   := vcpkg-rebuild-macos
_BUILD_TARGET    := build-mac
_EXIFTOOL_CHECK  := exiftool-check-macos
endif

# Package configuration
VCPKG_PACKAGES = \
	libraw[6by9rpi,demosaic-pack-gpl2,demosaic-pack-gpl3,dng-lossy,dngsdk,rawspeed,rawspeed3,x3ftools] \
	tiff[cxx,jpeg,lerc,libdeflate,lzma,webp,zip,zstd]

# ExifTool configuration
EXIFTOOL_VERSION   := 13.55
EXIFTOOL_SF_BASE   := https://sourceforge.net/projects/exiftool/files

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
build-mac: exiftool-check-macos
	wails build -platform darwin/universal

build-windows: export PKG_CONFIG_PATH := $(PKG_CONFIG_WINDOWS)
build-windows: exiftool-check-windows
	wails build -platform windows/amd64

dev: export PKG_CONFIG_PATH := $(VCPKG_INSTALLED)/$(TRIPLET)/lib/pkgconfig
dev: $(_EXIFTOOL_CHECK)
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
		--triplet=$(TRIPLET_ARM64) \
		--recurse

vcpkg-install-macos-x64:
	$(VCPKG) install $(VCPKG_PACKAGES) \
		--overlay-ports=$(OVERLAY_PORTS) \
		--overlay-triplets=$(OVERLAY_TRIPLETS) \
		--triplet=$(TRIPLET_X64_MAC) \
		--recurse

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
	$(VCPKG) remove libraw tiff adobe-dng-sdk rawspeed3 --triplet=$(TRIPLET_ARM64) --recurse
	$(VCPKG) remove libraw tiff adobe-dng-sdk rawspeed3 --triplet=$(TRIPLET_X64_MAC) --recurse
	rm -rf $(INSTALLED_UNIVERSAL)

vcpkg-rebuild-macos: vcpkg-clean-macos vcpkg-install-macos-universal

# ── vcpkg Windows targets ────────────────────────────────────────

vcpkg-install-windows:
	$(VCPKG) install $(VCPKG_PACKAGES) \
		--overlay-ports=$(OVERLAY_PORTS) \
		--overlay-triplets=$(OVERLAY_TRIPLETS) \
		--triplet=$(TRIPLET_WINDOWS) \
		--recurse

vcpkg-clean-windows:
	$(VCPKG) remove libraw tiff adobe-dng-sdk rawspeed3 --triplet=$(TRIPLET_WINDOWS) --recurse

vcpkg-rebuild-windows: vcpkg-clean-windows vcpkg-install-windows

# ── ExifTool download targets ───────────────────────────────────

# Windows: use PowerShell script (works in cmd.exe / PowerShell)
exiftool-check-windows exiftool-download-windows:
	powershell -ExecutionPolicy Bypass -File scripts/download-exiftool.ps1 -Version $(EXIFTOOL_VERSION)

# macOS: use bash commands (native on macOS)
_EXIFTOOL_MAC_VER := $(shell cat third-party/macos-universal/.exiftool-version 2>/dev/null)

ifneq ($(strip $(_EXIFTOOL_MAC_VER)),$(EXIFTOOL_VERSION))
exiftool-check-macos:
	$(MAKE) exiftool-download-macos
else
exiftool-check-macos:
	@echo "ExifTool $(EXIFTOOL_VERSION) already present for macOS"
endif

exiftool-download-macos:
	@echo "Downloading ExifTool $(EXIFTOOL_VERSION) for macOS..."
	@rm -rf third-party/macos-universal
	@mkdir -p third-party/macos-universal
	curl -L -o /tmp/exiftool-mac.tar.gz \
		"$(EXIFTOOL_SF_BASE)/Image-ExifTool-$(EXIFTOOL_VERSION).tar.gz/download"
	tar xzf /tmp/exiftool-mac.tar.gz -C /tmp
	cp /tmp/Image-ExifTool-$(EXIFTOOL_VERSION)/exiftool third-party/macos-universal/exiftool
	cp -r /tmp/Image-ExifTool-$(EXIFTOOL_VERSION)/lib third-party/macos-universal/lib
	chmod +x third-party/macos-universal/exiftool
	rm -rf /tmp/Image-ExifTool-$(EXIFTOOL_VERSION) /tmp/exiftool-mac.tar.gz
	@echo "$(EXIFTOOL_VERSION)" > third-party/macos-universal/.exiftool-version
	@echo "ExifTool $(EXIFTOOL_VERSION) downloaded for macOS"

exiftool-download:
ifeq ($(OS),Windows_NT)
	$(MAKE) exiftool-check-windows
else
	$(MAKE) exiftool-check-macos
endif

exiftool-clean:
	rm -rf third-party/windows-x64 third-party/macos-universal
