English | [简体中文](./README_zh-CN.md)

# OPEN MAKE TIFF

![](./doc/screenshot.png)

## About

`open make tiff` is an open-source alternative to [MakeTiff](https://www.colorperfect.com/MakeTiff/).

## Principle

`open make tiff`  utilizes three applications to complete the conversion from RAW to TIFF:
- Adobe DNG Converter:  Identifies the camera and performs bayer interpolation
- Libraw: Generates a linear TIFF without color processing
- exiftool: Copies EXIF metadata