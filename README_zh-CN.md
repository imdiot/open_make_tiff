[English](./README.md) | 简体中文

# OPEN MAKE TIFF

![](./doc/screenshot.png)

## 关于

`open make tiff` 是一个 [MakeTiff](https://www.colorperfect.com/MakeTiff/) 的开源替代品.

## 功能

1. 16bit 线性伽马 tiff
2. 无任何色彩调整

## 原理

`open make tiff` 调用三个应用程序来完成 RAW 到 TIFF 的转换:
- [Adobe DNG Converter](https://helpx.adobe.com/camera-raw/using/adobe-dng-converter.html)(可选，用户自行安装): 识别相机并进行拜耳差值，如未安装，则由 Libraw 进行。
- Libraw: 生成无色彩处理的线性 TIFF
- ExifTool: 复制 EXIF 信息
