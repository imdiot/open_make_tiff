English | [简体中文](./README_zh-CN.md)

# OPEN MAKE TIFF

![](./doc/screenshot.png)

## About

`open make tiff` is an open-source alternative to [MakeTiff](https://www.colorperfect.com/MakeTiff/).

## Features
1. 16-bit linear gamma TIFF
2. No color adjustments whatsoever

## Principle

`open make tiff`  utilizes three applications to complete the conversion from RAW to TIFF:
- [Adobe DNG Converter](https://helpx.adobe.com/camera-raw/using/adobe-dng-converter.html)(Optional, Self-Installation):  Identifies the camera and performs bayer interpolation; if not installed, Libraw will handle these tasks.
- Libraw: Generates a linear TIFF without color processing
- exiftool: Copies EXIF metadata