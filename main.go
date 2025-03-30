package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jessevdk/go-flags"
	"golang.org/x/term"
	"gopkg.in/ini.v1"
)

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

func run(logger *slog.Logger, sourceDir string, targets string, arcBase string, arcFile string, encrypt bool, encFile string, rcloneDestination string, rcloneConfigFile string, preserveArchive bool) (err error) {
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
				defer func() {
					err = handleDeferError(err, func() error {
						return os.Remove(arcFile)
					})
				}()
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
			defer func() {
				err = handleDeferError(err, func() error {
					return os.Remove(encFile)
				})
			}()
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
