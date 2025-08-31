package runner

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"open-make-tiff/pkg/icc"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"

	"open-make-tiff/pkg/binary"
	"open-make-tiff/pkg/util"
)

type Config struct {
	EnableAdobeDNGConverter bool
	EnableSubfolder         bool
	Profile                 string
}

type Runner struct {
	cfg    Config
	logger *slog.Logger
}

func New(cfg Config) *Runner {
	return &Runner{
		cfg:    cfg,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (r *Runner) Run(ctx context.Context, src string) error {
	var err error
	src, err = filepath.Abs(src)
	if err != nil {
		return err
	}

	srcDir := filepath.Dir(src)
	dstDir := srcDir
	if r.cfg.EnableSubfolder {
		dstDir = filepath.Join(dstDir, "make_tiff")
	}
	srcFileExt := filepath.Ext(src)
	srcFilename := filepath.Base(src)
	srcFilenameWithOutExt := srcFilename[:len(srcFilename)-len(filepath.Ext(srcFilename))]

	var (
		token               string
		tmpFilepathLog      string
		tmpFilepathInitRaw  string
		tmpFilepathInitTIFF string
		tmpFilepathTIFF     string
		dstFilepathTIFF     string
	)

	defer func() {
		for _, f := range []string{tmpFilepathInitRaw, tmpFilepathInitTIFF, tmpFilepathTIFF} {
			if f != "" {
				_ = os.Remove(f)
			}
		}
		if err != nil {
			_ = os.Remove(dstFilepathTIFF)
		}
	}()

	for {
		u := uuid.New()
		token = hex.EncodeToString(u[:])
		tmpFilepathLog = filepath.Join(dstDir, fmt.Sprintf("%s_%s.log", srcFilenameWithOutExt, token))
		tmpFilepathInitRaw = filepath.Join(dstDir, fmt.Sprintf("%s_%s.init", srcFilenameWithOutExt, token))
		tmpFilepathInitTIFF = filepath.Join(dstDir, fmt.Sprintf("%s_%s.init.tiff", srcFilenameWithOutExt, token))
		tmpFilepathTIFF = filepath.Join(dstDir, fmt.Sprintf("%s_%s.tiff", srcFilenameWithOutExt, token))
		for _, f := range []string{tmpFilepathLog, tmpFilepathInitRaw, tmpFilepathInitTIFF, tmpFilepathTIFF} {
			_, err = os.Stat(f)
			if err == nil || !errors.Is(err, os.ErrNotExist) {
				continue
			}
		}
		break
	}

	if err = os.MkdirAll(dstDir, 0644); err != nil {
		return err
	}

	f, err := os.Create(tmpFilepathLog)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			r.logger.Error(err.Error())
		}
		_ = f.Close()
		if err == nil {
			_ = os.Remove(tmpFilepathLog)
		}
	}()
	r.logger = slog.New(slog.NewTextHandler(f, nil))

	dstFilenameWithOutExt := srcFilenameWithOutExt
	for i := 0; ; i++ {
		if i != 0 {
			dstFilenameWithOutExt = fmt.Sprintf("%s_%d", srcFilenameWithOutExt, i)
		}
		dstFilepathTIFF = filepath.Join(dstDir, fmt.Sprintf("%s.tiff", dstFilenameWithOutExt))
		_, err = os.Stat(dstFilepathTIFF)
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			continue
		}
		break
	}
	r.logger.Info("src filepath: ", src)
	r.logger.Info("dst tiff filepath: ", dstFilepathTIFF)

	r.logger.Info("tmp adobe dng filepath: ", tmpFilepathInitRaw)
	r.logger.Info("tmf tiff filepath: ", tmpFilepathInitTIFF)

	if srcFileExt == ".fff" {
		err = r.runTiffcp(ctx, src, tmpFilepathTIFF)
		if err != nil {
			return err
		}
	} else {
		if r.cfg.EnableAdobeDNGConverter && util.EnableAdobeDNGConverter() {
			err = r.runAdobeDNGConverter(ctx, src, tmpFilepathInitRaw)
			if err != nil {
				r.logger.Warn(err.Error())
				err = nil
			}
		} else {
			err = r.runCopyFile(src, tmpFilepathInitRaw)
			if err != nil {
				return err
			}
		}

		if err = r.runCleanExif(ctx, tmpFilepathInitRaw); err != nil {
			return err
		}
		err = r.runDcrawEmuConvert(ctx, tmpFilepathInitRaw, tmpFilepathInitTIFF)
		if err != nil {
			return err
		}
		_ = os.Remove(tmpFilepathInitRaw)

		err = r.runTiffcp(ctx, tmpFilepathInitTIFF, tmpFilepathTIFF)
		if err != nil {
			return err
		}
		_ = os.Remove(tmpFilepathInitTIFF)
	}

	err = r.runCopyExif(ctx, src, tmpFilepathTIFF)
	if err != nil {
		return err
	}

	if err = r.runInsertICC(ctx, tmpFilepathTIFF, r.cfg.Profile); err != nil {
		return err
	}

	if err = os.Rename(tmpFilepathTIFF, dstFilepathTIFF); err != nil {
		return err
	}

	return nil
}

func (r *Runner) runCopyFile(src, dst string) error {
	srcFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = srcFile.Close()
	}()

	_, err = os.Stat(dst)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s already exists", dst)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = dstFile.Close()
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

func (r *Runner) runTiffcp(ctx context.Context, src string, dst string) error {
	tiffcpExecutable, err := util.GetTiffcpExecutable()
	if err != nil {
		return err
	}

	cfg := binary.Config{
		Executable: tiffcpExecutable,
	}
	cfg.Args = []string{
		"-,=%",
		fmt.Sprintf("%s%%0", src),
		dst,
	}
	r.logger.Info("run tiffcp: ", cfg.Executable, cfg.Args)
	if _, err := binary.New(cfg).Run(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Runner) runAdobeDNGConverter(ctx context.Context, src string, dst string) error {
	cfg := binary.Config{
		Executable: util.GetAdobeDNGConverterExecutable(),
	}
	cfg.Args = []string{
		"-c", "-u", "-l", "-p0",
		"-d", filepath.Dir(dst),
		"-o", filepath.Base(dst),
		src,
	}
	r.logger.Info("run adobe dng converter: ", cfg.Executable, cfg.Args)
	_, err := binary.New(cfg).Run(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *Runner) runDcrawEmuConvert(ctx context.Context, src string, dst string) error {
	dcrawEmuExecutable, err := util.GetDcrawEmuExecutable()
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		err = dstFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	args := []string{
		"-v", "-T", "-r", "1", "1", "1", "1", "-o", "0", "-4", "-Z", "-",
		src,
	}
	r.logger.Info("run dcraw_emu: ", dcrawEmuExecutable, args)
	cmd := exec.CommandContext(ctx, dcrawEmuExecutable, args...)
	cmd.SysProcAttr = util.GetSysProcAttr()
	cmd.Stdout = dstFile
	err = cmd.Run()
	if err = dstFile.Sync(); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *Runner) runCleanExif(ctx context.Context, src string) error {
	executable, err := util.GetExiftoolExecutable()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, executable, "-overwrite_original", "-tagsfromfile", src, "-ALL=", src)
	r.logger.Info("run clean exif IPTC: ", executable, cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

//func (r *Runner) runDeleteExifIPTC(ctx context.Context, src string) error {
//	exiv2Executable, err := util.GetExiv2Executable()
//	if err != nil {
//		return err
//	}
//
//	cmd := exec.CommandContext(ctx, exiv2Executable, "-di", src)
//	r.logger.Info("run delete exif IPTC: ", exiv2Executable, cmd.Args)
//	cmd.SysProcAttr = util.GetSysProcAttr()
//	err = cmd.Run()
//	if err != nil {
//		return err
//	}
//	return nil
//}

func (r *Runner) runCopyExif(ctx context.Context, src string, dst string) error {
	executable, err := util.GetExiftoolExecutable()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, executable, "-overwrite_original", "-tagsfromfile", src, "-ALL:ALL", dst)
	r.logger.Info("run copy exif: ", executable, cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

//func (r *Runner) runCopyExif(ctx context.Context, src string, dst string) error {
//	exiv2Executable, err := util.GetExiv2Executable()
//	if err != nil {
//		return err
//	}
//
//	cmdExtract := exec.CommandContext(ctx, exiv2Executable, "-ee-", src)
//	r.logger.Info("run copy exif: ", exiv2Executable, cmdExtract.Args)
//	cmdExtract.SysProcAttr = util.GetSysProcAttr()
//	var b bytes.Buffer
//	cmdExtract.Stdout = &b
//	err = cmdExtract.Run()
//	if err != nil {
//		return err
//	}
//
//	cmdInsert := exec.CommandContext(ctx, exiv2Executable, "-ie-", dst)
//	cmdInsert.SysProcAttr = util.GetSysProcAttr()
//	cmdInsert.Stdin = &b
//	err = cmdInsert.Run()
//	if err != nil {
//		return err
//	}
//	return nil
//}

func (r *Runner) runInsertICC(ctx context.Context, src string, name string) error {
	executable, err := util.GetExiftoolExecutable()
	if err != nil {
		return err
	}

	profile, ok := icc.Profiles[name]
	cmd := exec.CommandContext(ctx, executable, "-overwrite_original", "-ICC_Profile=", src)
	if ok {
		cmd = exec.CommandContext(ctx, executable, "-overwrite_original", "-ICC_Profile<=-", src)
		var b bytes.Buffer
		b.Write(profile.Data())
		cmd.Stdin = &b
	}
	r.logger.Info("run insert exif: ", executable, cmd.Args)
	fmt.Println("run insert exif: ", executable, cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

//func (r *Runner) runInsertICC(ctx context.Context, src string, name string) error {
//	exiv2Executable, err := util.GetExiv2Executable()
//	if err != nil {
//		return err
//	}
//
//	profile, ok := icc.Profiles[name]
//	if !ok {
//		return nil
//	}
//
//	cmd := exec.CommandContext(ctx, exiv2Executable, "-iC-", src)
//	r.logger.Info("run insert exif: ", exiv2Executable, cmd.Args)
//	cmd.SysProcAttr = util.GetSysProcAttr()
//	var b bytes.Buffer
//	b.Write(profile.Data())
//	cmd.Stdin = &b
//	err = cmd.Run()
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
