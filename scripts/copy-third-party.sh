#!/bin/bash
set -e

rm -rf ./open-make-tiff.app/Contents/MacOS/third-party
cp -r ../../third-party/macos-universal ./open-make-tiff.app/Contents/MacOS/third-party
