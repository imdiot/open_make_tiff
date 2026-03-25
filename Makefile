.PHONY: build-mac build-windows dev clean

build-mac:
	wails build -platform darwin/universal

build-windows:
	wails build -platform windows/amd64

dev:
	wails dev

clean:
	rm -rf build/bin/*
