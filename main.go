package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Luzifer/go-openssl/v4"
	"github.com/jessevdk/go-flags"
	"github.com/ncruces/go-sqlite3"
	"golang.org/x/term"
	"gopkg.in/ini.v1"

	_ "github.com/ncruces/go-sqlite3/embed"

	_ "github.com/rclone/rclone/backend/all"
	rcloneCmd "github.com/rclone/rclone/cmd"
	rcloneConfig "github.com/rclone/rclone/fs/config"
	rcloneConfigFile "github.com/rclone/rclone/fs/config/configfile"
	rcloneOperations "github.com/rclone/rclone/fs/operations"
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

	return nil
}

func (bv *bvCtx) backupFile(file string) error {
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
	defer f.Close()

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
			return nil
		}

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(bv.tw, f)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (bv *bvCtx) backupDb(name string) error {
	tmpname, err := tempFileName("db")
	if err != nil {
		return err
	}

	fname := filepath.Join(bv.srcdir, "db.sqlite3")

	info, err := os.Stat(fname)
	if err != nil {
		return err
	}

	uid, gid, mode := fileStatOf(info)

	db, err := sqlite3.Open(fname)
	if err != nil {
		return err
	}

	bv.logger.Debug("Vacuuming db info backup db file")
	err = db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpname))
	if err != nil {
		return err
	}

	if err = db.Close(); err != nil {
		return err
	}
	defer os.Remove(tmpname)

	f, err := os.Open(tmpname)
	if err != nil {
		return err
	}
	defer f.Close()

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

	return nil
}

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

func createBackup(logger *slog.Logger, filename string, srcdir string, arcbase string, bt *backupTargets) error {
	arc, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer arc.Close()

	gw, err := gzip.NewWriterLevel(arc, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

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

func encryptBackup(sourcefile string, destfile string, passphrase string) error {
	f, err := os.Open(sourcefile)
	if err != nil {
		return err
	}
	defer os.Remove(sourcefile)
	defer f.Close()

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
	defer fe.Close()

	_, err = fe.Write(enc)
	if err != nil {
		return err
	}

	return nil
}

func tempFileName(pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}

	name := f.Name()

	if err = f.Close(); err != nil {
		return "", err
	}

	if err = os.Remove(name); err != nil {
		return "", err
	}

	return name, nil
}

func getPassphrase() (string, error) {
	passphrase, ok := os.LookupEnv("BACKUP_PASSPHRASE")
	if !ok {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprint(os.Stderr, "Enter passphrase for a new backup file: ")
			binpass, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return "", err
			}
			fmt.Fprintln(os.Stderr)
			passphrase = string(binpass)
		}
	}

	return passphrase, nil
}

func logLevelForString(s string) (slog.Level, error) {
	var l slog.Level
	err := l.UnmarshalText([]byte(s))
	return l, err
}

type options struct {
	Output            string `short:"o" long:"output" description:"Output file (default: backup.tar.gz; backup.tar.gz.enc if --encrypt is set)"`
	Targets           string `short:"t" long:"targets" description:"Backup targets" default:"default"`
	Encrypt           bool   `short:"e" long:"encrypt" description:"Encrypt backup file. The passphrase can be set via BACKUP_PASSPHRASE environment variable, or interactively"`
	ArchiveBaseDir    string `short:"b" long:"archive-base-dir" description:"Base directory in archive" default:"data/"`
	RcloneDestination string `short:"r" long:"rclone-destination" description:"Rclone destination path.  If set, the backup file will be uploaded to the remote."`
	RcloneConfigFile  string `short:"c" long:"rclone-config-file" description:"Rclone config file."`
	PreserveArchive   bool   `short:"k" long:"preserve-archive" description:"Preserve archive file after uploading to rclone destination"`
	LogLevel          string `short:"l" long:"loglevel" description:"Log level" required:"true" default:"info"`

	Args struct {
		SourceDir string `positional-arg-name:"source-dir" description:"Source directory"`
	} `positional-args:"true" required:"true"`
}

func setup(opts *options) (runRclone bool, arcFile string, encFile string, err error) {
	const (
		defArcFile = "backup.tar.gz"
		defEncFile = "backup.tar.gz.enc"
	)

	if opts.RcloneDestination != "" {
		if opts.RcloneConfigFile == "" {
			return false, "", "", fmt.Errorf("rclone config file is required when rclone destination is set")
		}

		cfg, err := ini.Load(opts.RcloneConfigFile)
		if err != nil {
			return false, "", "", err
		}

		remotes := cfg.SectionStrings()
		remote := slices.IndexFunc(remotes, func(remote string) bool {
			return strings.HasPrefix(opts.RcloneDestination, remote+":")
		})
		if remote < 0 {
			return false, "", "", fmt.Errorf("no rclone remote setting found for destination '%s'", opts.RcloneDestination)
		}

		if !opts.PreserveArchive {
			if opts.Encrypt {
				arcFile, err = tempFileName("vwb")
				if err != nil {
					return false, "", "", err
				}

				file := ""
				if opts.Output == "" {
					_, file = filepath.Split(defEncFile)
				} else {
					_, file = filepath.Split(opts.Output)
				}
				encFile = filepath.Join(os.TempDir(), file)

				return true, arcFile, encFile, nil
			}

			file := ""
			if opts.Output == "" {
				_, file = filepath.Split(defArcFile)
			} else {
				_, file = filepath.Split(opts.Output)
			}
			arcFile = filepath.Join(os.TempDir(), file)

			return true, arcFile, "", nil
		}

		runRclone = true
	}

	if opts.Encrypt {
		arcFile, err = tempFileName("vwb")
		if err != nil {
			return false, "", "", err
		}

		if opts.Output == "" {
			encFile = defEncFile
		} else {
			encFile = opts.Output
		}

		return runRclone, arcFile, encFile, nil
	}

	if opts.Output == "" {
		arcFile = defArcFile
	} else {
		arcFile = opts.Output
	}

	return runRclone, arcFile, "", nil
}

func execRclone(logger *slog.Logger, configFile string, srcFile string, dest string) error {
	ctx := context.Background()

	if err := rcloneConfig.SetConfigPath(configFile); err != nil {
		return err
	}

	rcloneConfigFile.Install()

	rcloneCmd.SigInfoHandler()

	fsrc, srcFileName, fdst := rcloneCmd.NewFsSrcFileDst([]string{srcFile, dest})

	logger.Debug(fmt.Sprintf("fsrc: %s, srcFileName: %s, fdst: %s", fsrc, srcFileName, fdst))

	return rcloneOperations.CopyFile(ctx, fdst, fsrc, srcFileName, srcFileName)
}

func run(logger *slog.Logger, sourceDir string, targets string, arcBase string, arcFile string, encrypt bool, encFile string, rcloneDestination string, rcloneConfigFile string, preserveArchive bool) error {
	bt, err := backupTargetsFromString(targets)
	if err != nil {
		return err
	}

	logger.Info("Creating backup file")
	err = createBackup(logger, arcFile, sourceDir, arcBase, &bt)
	if err != nil {
		return err
	}

	if !encrypt {
		logger.Info(fmt.Sprintf("Backup file created: %s", arcFile))

		if rcloneDestination != "" {
			if !preserveArchive {
				defer os.Remove(arcFile)
			}

			logger.Info(fmt.Sprintf("Uploading backup file to %s", rcloneDestination))

			err = execRclone(logger, rcloneConfigFile, arcFile, rcloneDestination)
			if err != nil {
				return err
			}

			logger.Info(fmt.Sprintf("Backup file uploaded to %s", rcloneDestination))
		}

		return nil
	}

	logger.Info("Encrypting backup file")

	passphrase, err := getPassphrase()
	if err != nil {
		return err
	}

	err = encryptBackup(arcFile, encFile, passphrase)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("Encrypted backup file created: %s", encFile))

	if rcloneDestination != "" {
		if !preserveArchive {
			defer os.Remove(encFile)
		}

		logger.Info(fmt.Sprintf("Uploading encrypted backup file to %s", rcloneDestination))

		err = execRclone(logger, rcloneConfigFile, encFile, rcloneDestination)
		if err != nil {
			return err
		}

		logger.Info(fmt.Sprintf("Encrypted backup file uploaded to %s", rcloneDestination))
	}

	return nil
}

func main() {
	var opts options
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	logLevel, err := logLevelForString(opts.LogLevel)
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	runRclone, arcFile, encFile, err := setup(&opts)
	if err != nil {
		panic(err)
	}

	logger.Debug(fmt.Sprintf("runRclone: %v, arcFile: %s, encFile: %s\n", runRclone, arcFile, encFile))

	arcBase := opts.ArchiveBaseDir
	_, abf := filepath.Split(arcBase)
	if abf != "" {
		arcBase += "/" // FIXME?
	}

	rcloneDestination := ""
	rcloneConfigFile := ""
	preserveArchive := false
	if runRclone {
		rcloneDestination = opts.RcloneDestination
		rcloneConfigFile = opts.RcloneConfigFile
		preserveArchive = opts.PreserveArchive
	}

	err = run(logger, opts.Args.SourceDir, opts.Targets, arcBase, arcFile, opts.Encrypt, encFile, rcloneDestination, rcloneConfigFile, preserveArchive)
	if err != nil {
		panic(err)
	}
}
