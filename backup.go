package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/Luzifer/go-openssl/v4"
)

type backupTargets struct {
	Every       bool
	Db          bool
	Attachments bool
	Key         bool
	Config      bool
	Sends       bool
	IconCache   bool
}

func backupTargetsFromString(s string) (backupTargets, error) {
	var bt backupTargets
	var unsupported []string

	var tokens = strings.Split(s, ",")
	if s == "" {
		tokens = []string{"default"}
	}

	for _, token := range tokens {
		switch token {
		case "every":
			bt.Every = true
		case "all":
			bt.Db = true
			bt.Attachments = true
			bt.Key = true
			bt.Config = true
			bt.Sends = true
			bt.IconCache = true
		case "recommended":
			bt.Db = true
			bt.Attachments = true
			bt.Config = true
			bt.Key = true
		case "default":
			bt.Db = true
			bt.Attachments = true
			bt.Config = true
		case "db":
			bt.Db = true
		case "attachments":
			bt.Attachments = true
		case "key":
			bt.Key = true
		case "config":
			bt.Config = true
		case "sends":
			bt.Sends = true
		case "icon_cache":
			bt.IconCache = true
		default:
			unsupported = append(unsupported, token)
		}
	}

	if len(unsupported) > 0 {
		return bt, fmt.Errorf("unsupported backup targets: %s", strings.Join(unsupported, ", "))
	}

	return bt, nil
}

func createBackup(logger *slog.Logger, filename string, srcdir string, arcbase string, bt *backupTargets) (err error) {
	arc, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = handleDeferError(err, func() error {
			return arc.Close()
		})
	}()

	gw, err := gzip.NewWriterLevel(arc, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer func() {
		err = handleDeferError(err, func() error {
			return gw.Close()
		})
	}()

	tw := tar.NewWriter(gw)
	defer func() {
		err = handleDeferError(err, func() error {
			return tw.Close()
		})
	}()

	bv := bvCtx{
		logger:  logger,
		tw:      tw,
		srcdir:  srcdir,
		arcbase: arcbase,
	}

	if bt.Every {
		bv.logger.Info("Backing up everything except db")
		if err := bv.backupDir("", true); err != nil {
			return err
		}

		bv.logger.Info("Backing up db")
		if err := bv.backupDb("db.sqlite3"); err != nil {
			return err
		}
	} else {
		bv.logger.Info("Backing up root directory")
		if err := bv.backupRoot(); err != nil {
			return err
		}

		if bt.Attachments {
			bv.logger.Info("Backing up attachments")
			if err := bv.backupDir("attachments", false); err != nil {
				return err
			}
		}

		if bt.Key {
			bv.logger.Info("Backing up rsa keys")
			if err := bv.backupFilePattern("rsa_key.*"); err != nil {
				return err
			}
		}

		if bt.Config {
			bv.logger.Info("Backing up config")
			if err := bv.backupFile("config.json"); err != nil {
				return err
			}
		}

		if bt.Sends {
			bv.logger.Info("Backing up sends")
			if err := bv.backupDir("sends", false); err != nil {
				return err
			}
		}

		if bt.IconCache {
			bv.logger.Info("Backing up icon cache")
			if err := bv.backupDir("icon_cache", false); err != nil {
				return err
			}
		}

		if bt.Db {
			bv.logger.Info("Backing up db")
			if err := bv.backupDb("db.sqlite3"); err != nil {
				return err
			}
		}
	}

	return nil
}

func encryptBackup(sourcefile string, destfile string, passphrase string) (err error) {
	f, err := os.Open(sourcefile)
	if err != nil {
		return err
	}
	defer func() {
		err = handleDeferError(err, func() error {
			return os.Remove(sourcefile)
		})
	}()
	defer func() {
		err = handleDeferError(err, func() error {
			return f.Close()
		})
	}()

	bin, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	o := openssl.New()
	enc, err := o.EncryptBinaryBytes(passphrase, bin, openssl.PBKDF2SHA256)
	if err != nil {
		return err
	}

	fe, err := os.Create(destfile)
	if err != nil {
		return err
	}
	defer func() {
		err = handleDeferError(err, func() error {
			return fe.Close()
		})
	}()

	_, err = fe.Write(enc)
	if err != nil {
		return err
	}

	return nil
}
