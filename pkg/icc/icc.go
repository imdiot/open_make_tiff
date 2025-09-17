package icc

import _ "embed"

var (
	//go:embed AdobeRGB1998.icc
	_AdobeRGB1998 []byte
	//go:embed ITU-2020.icc
	_BT2020 []byte
	//go:embed DisplayP3.icc
	_DisplayP3 []byte
	//go:embed HasselbladRGB.icc
	_HasselbladRGB []byte
	//go:embed ProPhoto.icm
	_ProPhoto []byte
	//go:embed sRGB.icc
	_sRGB []byte
)

var Profiles = map[string]*Profile{
	"Adobe RGB 1998": {name: "Adobe RGB 1998", data: _AdobeRGB1998},
	"BT.2020":        {name: "BT.2020", data: _BT2020},
	"Display P3":     {name: "Display P3", data: _DisplayP3},
	"Hasselblad RGB": {name: "Hasselblad RGB", data: _HasselbladRGB},
	"ProPhoto":       {name: "ProPhoto", data: _ProPhoto},
	"sRGB":           {name: "sRGB", data: _sRGB},
}

type Profile struct {
	name string
	data []byte
}

func (p *Profile) Name() string {
	return p.name
}

func (p *Profile) Data() []byte {
	return p.data
}
