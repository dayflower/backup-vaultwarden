package main

import (
	"context"
	"fmt"
	"log/slog"

	rcloneCmd "github.com/rclone/rclone/cmd"
	rcloneConfig "github.com/rclone/rclone/fs/config"
	rcloneConfigFile "github.com/rclone/rclone/fs/config/configfile"
	rcloneOperations "github.com/rclone/rclone/fs/operations"

	_ "github.com/rclone/rclone/backend/all"
)

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
