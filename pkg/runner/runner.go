package runner

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"

	"open-make-tiff/pkg/icc"
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

	if err = os.MkdirAll(dstDir, 0755); err != nil {
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

		//if err = r.runCleanExif(ctx, tmpFilepathInitRaw); err != nil {
		//	return err
		//}

		if err = r.runDcrawEmuConvert(ctx, tmpFilepathInitRaw, tmpFilepathInitTIFF); err != nil {
			return err
		}
		_ = os.Remove(tmpFilepathInitRaw)

		if err = r.runTiffcp(ctx, tmpFilepathInitTIFF, tmpFilepathTIFF); err != nil {
			return err
		}
		_ = os.Remove(tmpFilepathInitTIFF)
	}

	if err = r.runCopyExif(ctx, src, tmpFilepathTIFF); err != nil {
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
	executable, err := util.GetTiffcpExecutable()
	if err != nil {
		return err
	}

	args := []string{
		"-,=%",
		fmt.Sprintf("%s%%0", src),
		dst,
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	r.logger.Info("run tiffcp: ", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
}

func (r *Runner) runAdobeDNGConverter(ctx context.Context, src string, dst string) error {
	executable := util.GetAdobeDNGConverterExecutable()
	args := []string{
		"-c", "-u", "-l", "-p0",
		"-d", filepath.Dir(dst),
		"-o", filepath.Base(dst),
		src,
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	r.logger.Info("run adobe dng converter: ", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
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
		_ = dstFile.Close()
	}()

	args := []string{
		"-T", "-r", "1", "1", "1", "1", "-o", "0", "-4", "-Z", "-",
		filepath.Base(src),
	}
	cmd := exec.CommandContext(ctx, dcrawEmuExecutable, args...)
	r.logger.Info("run dcraw_emu: ", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	cmd.Dir = filepath.Dir(src)
	cmd.Stdout = dstFile
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return err
	}
	if stderr.String() != "" {
		return fmt.Errorf(stderr.String())
	}
	return nil
}

func (r *Runner) runCleanExif(ctx context.Context, src string) error {
	executable, err := util.GetExiftoolExecutable()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, executable, "-overwrite_original", "-tagsfromfile", src, "-ALL=", src)
	r.logger.Info("run clean exif: ", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
}

func (r *Runner) runCopyExif(ctx context.Context, src string, dst string) error {
	executable, err := util.GetExiftoolExecutable()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, executable, "-overwrite_original", "-tagsfromfile", src, "-ALL:ALL", dst)
	r.logger.Info("run copy exif: ", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
}

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
	r.logger.Info("run insert exif: ", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
}
