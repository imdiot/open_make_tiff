package icc

import _ "embed"

var (
	//go:embed AdobeRGB1998.icc
	_AdobeRGB1998 []byte
	//go:embed BT2020.icc
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
	"AdobeRGB1998":  {Name: "Adobe RGB 1998", Data: _AdobeRGB1998},
	"BT2020":        {Name: "BT.2020", Data: _BT2020},
	"DisplayP3":     {Name: "Display P3", Data: _DisplayP3},
	"HasselbladRGB": {Name: "Hasselblad RGB", Data: _HasselbladRGB},
	"ProPhoto":      {Name: "ProPhoto", Data: _ProPhoto},
	"sRGB":          {Name: "sRGB", Data: _sRGB},
}

type Profile struct {
	Name string
	Data []byte
}
