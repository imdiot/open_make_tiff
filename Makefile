# OpenMakeTiff build orchestration: wails app + vcpkg deps (libraw/tiff) + ExifTool.
# Cross-platform (macOS/Windows). GNU make 3.81 compatible (no .RECIPEPREFIX, no !=).

.PHONY: build build-mac build-windows dev clean test vcpkg-bootstrap vcpkg-check \
        package package-mac package-windows \
        vcpkg-install vcpkg-install-macos-arm64 vcpkg-install-macos-x64 \
        vcpkg-install-macos-universal vcpkg-install-windows \
        vcpkg-clean vcpkg-clean-macos vcpkg-clean-windows \
        vcpkg-rebuild vcpkg-rebuild-macos vcpkg-rebuild-windows \
        exiftool-check exiftool-download exiftool-download-macos exiftool-download-windows \
        exiftool-check-macos exiftool-check-windows exiftool-clean

# ── Platform detection ───────────────────────────────────────────
# Set one host flag + the vcpkg binary/root only. Target dispatch is expressed
# via $(if) in "Dispatch variables" — never inline functions in target lines.
# VCPKG_REF must match vcpkg-configuration.json baseline (= tag 2026.03.18)
VCPKG_REF   := c3867e714dd3a51c272826eea77267876517ed99
VCPKG_ROOT  := $(CURDIR)/third-party/vcpkg-root
ifeq ($(OS),Windows_NT)
IS_WINDOWS := 1
VCPKG      := $(VCPKG_ROOT)/vcpkg.exe
VCPKG_BOOT := powershell -ExecutionPolicy Bypass -File scripts/download-vcpkg.ps1 -Ref $(VCPKG_REF)
else
IS_MAC     := 1
VCPKG      := $(VCPKG_ROOT)/vcpkg
VCPKG_BOOT := bash scripts/download-vcpkg.sh $(VCPKG_REF)
endif

# ── ExifTool configuration ───────────────────────────────────────
# (vcpkg deps are declared in vcpkg.json — manifest mode)
EXIFTOOL_VERSION := 13.55
EXIFTOOL_SF_BASE := https://sourceforge.net/projects/exiftool/files

OVERLAY_PORTS    = third-party/vcpkg-overlay/ports
OVERLAY_TRIPLETS = third-party/vcpkg-overlay/triplets
VCPKG_INSTALLED  = $(CURDIR)/vcpkg_installed

# ── Triplets & derived paths ─────────────────────────────────────
TRIPLET_ARM64     := arm64-osx-release
TRIPLET_X64_MAC   := x64-osx-release
TRIPLET_UNIVERSAL := universal-osx-release
TRIPLET_WINDOWS   := x64-mingw-static-release

INSTALLED_ARM64     := $(VCPKG_INSTALLED)/$(TRIPLET_ARM64)
INSTALLED_X64_MAC   := $(VCPKG_INSTALLED)/$(TRIPLET_X64_MAC)
INSTALLED_UNIVERSAL := $(VCPKG_INSTALLED)/$(TRIPLET_UNIVERSAL)
INSTALLED_WINDOWS   := $(VCPKG_INSTALLED)/$(TRIPLET_WINDOWS)

PKG_CONFIG_MAC     := $(INSTALLED_UNIVERSAL)/lib/pkgconfig
PKG_CONFIG_WINDOWS := $(INSTALLED_WINDOWS)/lib/pkgconfig

# Native host triplet — drives `dev`'s PKG_CONFIG_PATH on the dev machine.
# Must stay platform-aware, or Windows `dev` would point at a nonexistent
# universal dir. Defined after the TRIPLET_* constants so the := resolves now.
TRIPLET := $(if $(IS_WINDOWS),$(TRIPLET_WINDOWS),$(TRIPLET_UNIVERSAL))

# ── Dispatch variables (host-platform → target) ──────────────────
_BUILD_TARGET   := $(if $(IS_MAC),build-mac,build-windows)
_PACKAGE_TARGET := $(if $(IS_MAC),package-mac,package-windows)
_EXIFTOOL_CHECK := $(if $(IS_MAC),exiftool-check-macos,exiftool-check-windows)
_VCPKG_INSTALL  := $(if $(IS_MAC),vcpkg-install-macos-universal,vcpkg-install-windows)
_VCPKG_CLEAN    := $(if $(IS_MAC),vcpkg-clean-macos,vcpkg-clean-windows)
_VCPKG_REBUILD  := $(if $(IS_MAC),vcpkg-rebuild-macos,vcpkg-rebuild-windows)

# ── Recipe templates (canned recipes) ────────────────────────────
# One install recipe shared by every triplet — only --triplet differs. Body
# lines MUST be tab-indented (3.81 has no .RECIPEPREFIX); the $(call) arg
# MUST have no leading space, or --triplet= gets one and vcpkg rejects it.
define vcpkg-install-recipe
	$(VCPKG) install \
		--vcpkg-root=$(VCPKG_ROOT) \
		--overlay-ports=$(OVERLAY_PORTS) \
		--overlay-triplets=$(OVERLAY_TRIPLETS) \
		--triplet=$(1) \
		--x-manifest-root=$(CURDIR) \
		--x-install-root=$(VCPKG_INSTALLED)
endef

# ── Build ────────────────────────────────────────────────────────
build: $(_BUILD_TARGET)

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

test: export PKG_CONFIG_PATH := $(PKG_CONFIG_MAC)
test:
	go test ./internal/... ./pkg/...

# ── Packaging ────────────────────────────────────────────────────
package: $(_PACKAGE_TARGET)

# Read productVersion from wails.json for the dmg filename.
APP_VERSION = $(shell grep -o '"productVersion"[[:space:]]*:[[:space:]]*"[^"]*"' wails.json | sed 's/.*"\([0-9.]*\)"$$/\1/')

# Build, then package the .app into a drag-to-Applications DMG. Installing by
# dragging into /Applications avoids the App Translocation that a quarantined
# zip extract triggers (which left the app bouncing in the Dock with no window).
package-mac: build-mac
	@rm -f build/bin/OpenMakeTiff-*-macos-universal.dmg
	@tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	find build/bin/OpenMakeTiff.app -name '.DS_Store' -delete; \
	cp -R build/bin/OpenMakeTiff.app "$$tmp/"; \
	ln -s /Applications "$$tmp/Applications"; \
	hdiutil create -volname "Open Make Tiff" -srcfolder "$$tmp" \
		-ov -format UDZO build/bin/OpenMakeTiff-$(APP_VERSION)-macos-universal.dmg >/dev/null; \
	echo "✓ packaged: build/bin/OpenMakeTiff-$(APP_VERSION)-macos-universal.dmg"

# Build, then zip the exe + third-party/ into a portable archive. The two must
# stay siblings so the runtime finds exiftool at filepath.Dir(self)/third-party/.
package-windows: build-windows
	powershell -ExecutionPolicy Bypass -File scripts/package-windows.ps1

# ── vcpkg bootstrap ─────────────────────────────────────────────
# Check whether the vendored tool needs re-bootstrapping (lightweight:
# a shell comparison against the marker file). Upgrading VCPKG_REF
# re-triggers clone + bootstrap automatically.
.PHONY: vcpkg-check
vcpkg-check:
	@if [ ! -x "$(VCPKG)" ] || [ ! -f "$(VCPKG_ROOT)/.vcpkg-ref" ] || [ "$$(cat $(VCPKG_ROOT)/.vcpkg-ref)" != "$(VCPKG_REF)" ]; then \
		$(VCPKG_BOOT); \
	fi
vcpkg-bootstrap: vcpkg-check

# ── vcpkg dispatch ───────────────────────────────────────────────
vcpkg-install: $(_VCPKG_INSTALL)
vcpkg-clean:   $(_VCPKG_CLEAN)
vcpkg-rebuild: $(_VCPKG_REBUILD)

# ── vcpkg macOS ──────────────────────────────────────────────────
vcpkg-install-macos-arm64: vcpkg-check
	$(call vcpkg-install-recipe,$(TRIPLET_ARM64))

vcpkg-install-macos-x64: vcpkg-check
	$(call vcpkg-install-recipe,$(TRIPLET_X64_MAC))

# Merge arm64 + x64 into universal fat libraries.
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

# manifest mode: clean = drop the built install trees (re-install rebuilds them).
vcpkg-clean-macos:
	rm -rf $(INSTALLED_ARM64) $(INSTALLED_X64_MAC) $(INSTALLED_UNIVERSAL)

vcpkg-rebuild-macos: vcpkg-clean-macos vcpkg-install-macos-universal

# ── vcpkg Windows ────────────────────────────────────────────────
vcpkg-install-windows: vcpkg-check
	$(call vcpkg-install-recipe,$(TRIPLET_WINDOWS))

vcpkg-clean-windows:
	rm -rf $(INSTALLED_WINDOWS)

vcpkg-rebuild-windows: vcpkg-clean-windows vcpkg-install-windows

# ── ExifTool ─────────────────────────────────────────────────────
# Windows: one PowerShell script covers both check and download (it decides
# internally whether the requested version is already on disk).
exiftool-check-windows exiftool-download-windows:
	powershell -ExecutionPolicy Bypass -File scripts/download-exiftool.ps1 -Version $(EXIFTOOL_VERSION)

# macOS: native bash. Re-download only when the recorded version differs.
_EXIFTOOL_MAC_VER := $(shell cat third-party/macos-universal/.exiftool-version 2>/dev/null)

ifneq ($(strip $(_EXIFTOOL_MAC_VER)),$(EXIFTOOL_VERSION))
exiftool-check-macos: exiftool-download-macos
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

# Cross-platform check / download entry points: pure prereq dispatch, no sub-make.
# download keeps its original "= check" semantics (re-downloads only when the
# recorded version differs), so check and download are interchangeable.
exiftool-check:   $(_EXIFTOOL_CHECK)
exiftool-download: $(_EXIFTOOL_CHECK)

exiftool-clean:
	rm -rf third-party/windows-x64 third-party/macos-universal
