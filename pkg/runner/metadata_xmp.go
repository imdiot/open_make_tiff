package runner

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// XMP namespace prefix → URI.
var xmpNS = map[string]string{
	"aux":    "http://ns.adobe.com/exif/1.0/aux/",
	"exifEX": "http://cipa.jp/exif/1.0/",
	"dc":     "http://purl.org/dc/elements/1.1/",
	"lr":     "http://ns.adobe.com/lightroom/1.0/",
	"mwg-kw": "http://www.metadataworkinggroup.com/schemas/keywords/",
	"crs":    "http://ns.adobe.com/camera-raw-settings/1.0/",
}

// knownArrayTags lists XMP tag names that should be serialized as rdf:Bag arrays.
// All other tags are treated as scalar values.
var knownArrayTags = map[string]bool{
	"Subject":             true, // dc:Subject
	"HierarchicalSubject": true, // lr:HierarchicalSubject
	"Keywords":            true, // mwg-kw:Keywords
}

func (em *ExtractedMetadata) BuildXMPacket() ([]byte, error) {
	type elem struct {
		prefix  string
		key     string
		value   string
		isArray bool
	}

	var elements []elem
	usedNS := make(map[string]string)
	for key, ti := range em.XMP {
		group, name := SplitGroupKey(key)
		prefix := strings.TrimPrefix(group, "XMP-")
		uri, ok := xmpNS[prefix]
		if !ok || ti.Val == "" {
			continue
		}
		elements = append(elements, elem{
			prefix: prefix, key: name,
			value: ti.Val, isArray: knownArrayTags[name],
		})
		usedNS[prefix] = uri
	}
	if len(elements) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)

	if err := enc.EncodeToken(xml.ProcInst{Target: "xpacket", Inst: []byte("begin=\"\xEF\xBB\xBF\" id=\"W5M0MpCehiHzreSzNTczkc9d\"")}); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}
	if err := enc.EncodeToken(xml.CharData("\n")); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}

	meta := xml.StartElement{
		Name: xml.Name{Local: "x:xmpmeta"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:x"}, Value: "adobe:ns:meta/"}},
	}
	if err := enc.EncodeToken(meta); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}
	if err := enc.EncodeToken(xml.CharData("\n ")); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}

	rdf := xml.StartElement{
		Name: xml.Name{Local: "rdf:RDF"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:rdf"}, Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#"}},
	}
	if err := enc.EncodeToken(rdf); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}
	if err := enc.EncodeToken(xml.CharData("\n  ")); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}

	desc := xml.StartElement{
		Name: xml.Name{Local: "rdf:Description"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "rdf:about"}, Value: ""}},
	}
	for prefix, uri := range usedNS {
		desc.Attr = append(desc.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:" + prefix}, Value: uri})
	}
	if err := enc.EncodeToken(desc); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}

	for _, e := range elements {
		name := e.prefix + ":" + e.key
		var err error
		if e.isArray {
			var items []string
			for part := range strings.SplitSeq(e.value, ", ") {
				if part != "" {
					items = append(items, part)
				}
			}
			if len(items) == 0 {
				continue
			}
			tag := xml.StartElement{Name: xml.Name{Local: name}}
			bag := xml.StartElement{Name: xml.Name{Local: "rdf:Bag"}}
			if err := enc.EncodeToken(tag); err != nil {
				return nil, fmt.Errorf("xmp element %s: %w", name, err)
			}
			if err := enc.EncodeToken(bag); err != nil {
				return nil, fmt.Errorf("xmp element %s: %w", name, err)
			}
			for _, item := range items {
				if err := writeXMLElement(enc, "rdf:li", item); err != nil {
					return nil, fmt.Errorf("xmp element %s: %w", name, err)
				}
			}
			if err := enc.EncodeToken(bag.End()); err != nil {
				return nil, fmt.Errorf("xmp element %s: %w", name, err)
			}
			if err := enc.EncodeToken(tag.End()); err != nil {
				return nil, fmt.Errorf("xmp element %s: %w", name, err)
			}
		} else {
			err = writeXMLElement(enc, name, e.value)
		}
		if err != nil {
			return nil, fmt.Errorf("xmp element %s: %w", name, err)
		}
	}

	for _, tok := range []xml.Token{desc.End(), xml.CharData("\n  "), rdf.End(), xml.CharData("\n "), meta.End(), xml.CharData("\n")} {
		if err := enc.EncodeToken(tok); err != nil {
			return nil, fmt.Errorf("xmp: %w", err)
		}
	}
	if err := enc.EncodeToken(xml.ProcInst{Target: "xpacket", Inst: []byte("end=\"w\"")}); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}
	if err := enc.EncodeToken(xml.CharData("\n")); err != nil {
		return nil, fmt.Errorf("xmp: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("xmp flush: %w", err)
	}
	return buf.Bytes(), nil
}

func writeXMLElement(enc *xml.Encoder, name, text string) error {
	start := xml.StartElement{Name: xml.Name{Local: name}}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if err := enc.EncodeToken(xml.CharData(text)); err != nil {
		return err
	}
	return enc.EncodeToken(start.End())
}

