package metadata

import (
	"fmt"

	"open-make-tiff/pkg/golibtiff"
)

func (em *ExtractedMetadata) WriteIFD0(tf *golibtiff.TIFF) {
	em.writeGroup(tf, em.IFD0, skipIFD0IDs)
	if len(em.XMP) > 0 {
		packet, err := em.BuildXMPacket()
		if err != nil {
			em.logger.Debug("build XMP packet failed", "err", err)
			return
		}
		_ = tf.SetFieldByteSlice(golibtiff.TagXMP, packet)
	}
}

func (em *ExtractedMetadata) ReserveSubIFDs(tf *golibtiff.TIFF) error {
	if len(em.EXIF) > 0 {
		if err := tf.SetFieldUint64(golibtiff.TagEXIFIFD, 0); err != nil {
			return fmt.Errorf("reserve EXIF IFD offset: %w", err)
		}
	}
	if len(em.GPS) > 0 {
		if err := tf.SetFieldUint64(golibtiff.TagGPSIFD, 0); err != nil {
			return fmt.Errorf("reserve GPS IFD offset: %w", err)
		}
	}
	return nil
}

func (em *ExtractedMetadata) WriteSubIFDs(tf *golibtiff.TIFF) error {
	if err := tf.WriteDirectory(); err != nil {
		return fmt.Errorf("write IFD0: %w", err)
	}
	if len(em.EXIF) > 0 {
		if err := em.writeSubIFD(tf, tf.CreateEXIFDirectory, em.EXIF, golibtiff.TagEXIFIFD); err != nil {
			return err
		}
	}
	if len(em.GPS) > 0 {
		if err := em.writeSubIFD(tf, tf.CreateGPSDirectory, em.GPS, golibtiff.TagGPSIFD); err != nil {
			return err
		}
	}
	return nil
}

func (em *ExtractedMetadata) writeSubIFD(tf *golibtiff.TIFF, createFn func() error, tags map[string]TagInfo, ptrTag golibtiff.Tag) error {
	if err := tf.SetDirectory(0); err != nil {
		return fmt.Errorf("set dir 0: %w", err)
	}
	if err := createFn(); err != nil {
		return fmt.Errorf("create sub-IFD: %w", err)
	}
	em.writeGroup(tf, tags, nil)
	offset, err := tf.WriteCustomDirectory()
	if err != nil {
		return fmt.Errorf("write custom directory: %w", err)
	}
	if err := tf.SetDirectory(0); err != nil {
		return fmt.Errorf("set dir 0 after sub-IFD: %w", err)
	}
	if err := tf.SetFieldUint64(ptrTag, offset); err != nil {
		return fmt.Errorf("set sub-IFD pointer: %w", err)
	}
	return tf.WriteDirectory()
}

func (em *ExtractedMetadata) writeGroup(tf *golibtiff.TIFF, tags map[string]TagInfo, skip map[golibtiff.Tag]bool) {
	for _, ti := range tags {
		if err := em.writeTag(tf, ti, skip); err != nil {
			id := ti.TagID()
			em.logger.Debug("write tag failed", "id", id, "err", err)
		}
	}
}

func (em *ExtractedMetadata) writeTag(tf *golibtiff.TIFF, ti TagInfo, skipIDs map[golibtiff.Tag]bool) error {
	id := ti.TagID()
	if skipIDs != nil && skipIDs[id] {
		return nil
	}
	tag := golibtiff.Tag(id)

	ft := tf.GetFieldType(tag)
	if ft < 0 {
		return nil
	}

	wc := tf.FieldWriteCount(tag)
	val := ti.ExtractValue(ft, wc)
	if val == nil {
		return nil
	}

	return tf.SetFieldAny(tag, val)
}
