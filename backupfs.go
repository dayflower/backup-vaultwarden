package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ncruces/go-sqlite3"

	_ "github.com/ncruces/go-sqlite3/embed"
)

type bvCtx struct {
	logger  *slog.Logger
	tw      *tar.Writer
	srcdir  string
	arcbase string
}

func (bv *bvCtx) backupRoot() error {
	info, err := os.Stat(bv.srcdir)
	if err != nil {
		return err
	}

	uid, gid, mode := fileStatOf(info)

	if err = bv.tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     bv.arcbase,
		Mode:     mode,
		ModTime:  info.ModTime(),
		Size:     info.Size(),
		Uid:      uid,
		Gid:      gid,
	}); err != nil {
		return err
	}

	bv.logger.Debug(fmt.Sprintf("Backed up root dir %s", bv.arcbase))

	return nil
}

func (bv *bvCtx) backupFile(file string) (err error) {
	fname := filepath.Join(bv.srcdir, file)

	info, err := os.Stat(fname)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	uid, gid, mode := fileStatOf(info)

	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer func() {
		err = handleDeferError(err, func() error {
			return f.Close()
		})
	}()

	if err = bv.tw.WriteHeader(&tar.Header{
		Name:    filepath.Join(bv.arcbase, file),
		Mode:    mode,
		ModTime: info.ModTime(),
		Size:    info.Size(),
		Uid:     uid,
		Gid:     gid,
	}); err != nil {
		return err
	}

	_, err = io.Copy(bv.tw, f)
	if err != nil {
		return err
	}

	bv.logger.Debug(fmt.Sprintf("Backed up %s", file))

	return nil
}

func (bv *bvCtx) backupFilePattern(pattern string) error {
	matches, err := filepath.Glob(filepath.Join(bv.srcdir, pattern))
	if err != nil {
		return err
	}

	for _, match := range matches {
		bv.logger.Debug(fmt.Sprintf("Found %s", match))

		err = bv.backupFile(match)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bv *bvCtx) backupDir(dir string, skipDb bool) error {
	if _, err := os.Stat(filepath.Join(bv.srcdir, dir)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	err := filepath.Walk(filepath.Join(bv.srcdir, dir), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fname, err := filepath.Rel(bv.srcdir, p)
		if err != nil {
			return err
		}

		bv.logger.Debug(fmt.Sprintf("Found %s", fname))

		uid, gid, mode := fileStatOf(info)

		typeflag := byte(0)
		if info.IsDir() {
			typeflag = tar.TypeDir
		} else {
			if skipDb && strings.HasSuffix(fname, "db.sqlite3") {
				return nil
			}
		}

		if err = bv.tw.WriteHeader(&tar.Header{
			Typeflag: typeflag,
			Name:     filepath.Join(bv.arcbase, fname),
			Mode:     mode,
			ModTime:  info.ModTime(),
			Size:     info.Size(),
			Uid:      uid,
			Gid:      gid,
		}); err != nil {
			return err
		}

		if info.IsDir() {
			bv.logger.Debug(fmt.Sprintf("Backed up dir %s", fname))

			return nil
		}

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer func() {
			err = handleDeferError(err, func() error {
				return f.Close()
			})
		}()

		_, err = io.Copy(bv.tw, f)
		if err != nil {
			return err
		}

		bv.logger.Debug(fmt.Sprintf("Backed up %s", fname))

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (bv *bvCtx) backupDb(name string) (err error) {
	tmpname, err := tempFileName("db")
	if err != nil {
		return err
	}

	fname := filepath.Join(bv.srcdir, name)

	info, err := os.Stat(fname)
	if err != nil {
		return err
	}

	uid, gid, mode := fileStatOf(info)

	db, err := sqlite3.Open(fname)
	if err != nil {
		return err
	}

	bv.logger.Debug(fmt.Sprintf("Vacuuming db into backup db file %s", tmpname))
	err = db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpname))
	if err != nil {
		return err
	}

	if err = db.Close(); err != nil {
		return err
	}
	defer func() {
		err = handleDeferError(err, func() error {
			return os.Remove(tmpname)
		})
	}()

	f, err := os.Open(tmpname)
	if err != nil {
		return err
	}
	defer func() {
		err = handleDeferError(err, func() error {
			return f.Close()
		})
	}()

	fs, err := f.Stat()
	if err != nil {
		return err
	}

	if err = bv.tw.WriteHeader(&tar.Header{
		Name:    filepath.Join(bv.arcbase, name),
		Mode:    mode,
		ModTime: info.ModTime(),
		Size:    fs.Size(),
		Uid:     uid,
		Gid:     gid,
	}); err != nil {
		return err
	}

	_, err = io.Copy(bv.tw, f)
	if err != nil {
		return err
	}

	bv.logger.Debug(fmt.Sprintf("Backed up db %s", name))

	return nil
}
