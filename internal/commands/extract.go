package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/librun/ha-backup-tool/internal/extractor"
	"github.com/librun/ha-backup-tool/internal/flags"
	"github.com/librun/ha-backup-tool/internal/options"
	"github.com/librun/ha-backup-tool/internal/tarextractor"
)

var (
	ErrNotFullExtract = errors.New("please check that your backup files and emergency kit are correct")
)

// Extract - command for extract archive.
func Extract() *cli.Command {
	return &cli.Command{
		Name:    "extract",
		Aliases: []string{"unpack", "e", "u"},
		Usage:   "command for decrypt and extract one or more backups",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "backups",
				UsageText: "files for extract backup home assistant in tar format",
				Min:       1,
				Max:       -1,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    flags.ExtractInclude,
				Aliases: []string{"ic"},
				Usage:   "Include files",
			},
			&cli.StringFlag{
				Name:    flags.ExtractExclude,
				Aliases: []string{"ec"},
				Usage:   "Exclude files",
			},
			&cli.StringFlag{
				Name:    flags.ExtractOutput,
				Aliases: []string{"o"},
				Usage:   "Directory for unpack files",
			},
			&cli.BoolFlag{
				Name:  flags.ExtractSkipCreateLinks,
				Usage: "Skip create symlinks and hard links",
			},
		},
		Action: extractAction,
	}
}

// extractAction - command for extract backups.
func extractAction(_ context.Context, c *cli.Command) error {
	var fs = c.StringArgs("backups")

	ops, err := options.NewCmdExtractOptions(c)
	if err != nil {
		return err
	}

	if len(fs) == 0 {
		fmt.Println("\n⚠️  No files for extract.")

		return nil
	}

	ops.ExtractToSubDir = len(fs) > 1
	if ops.ExtractToSubDir && ops.OutputDir != "" {
		if _, errS := os.Stat(ops.OutputDir); os.IsNotExist(errS) {
			if err = os.Mkdir(ops.OutputDir, tarextractor.UnpackDirMod); err != nil {
				return err
			}
		}
	}

	fmt.Printf("📁 Found %s backup file(s) to process\n", fs)

	var s int
	var wg = sync.WaitGroup{}

	for _, f := range fs {
		wg.Add(1)

		go func() {
			if errF := extractActionFile(f, ops, &wg); errF == nil {
				s++
			}
		}()
	}

	wg.Wait()

	if s > 0 {
		fmt.Printf("\n✅ Successfully decrypted %v of %v backup file(s)!\n", s, len(fs))
		fmt.Println("You can find the decrypted files in the extracted directories.")
	} else {
		fmt.Println("\n⚠️ No files were successfully decrypted.")
	}

	if s == 0 || s != len(fs) {
		return ErrNotFullExtract
	}

	return nil
}

func extractActionFile(f string, ops *options.CmdExtractOptions, wg *sync.WaitGroup) error {
	defer wg.Done()

	if err := extractor.ValidateTarFile(f); err != nil {
		if !ops.Verbose {
			fmt.Printf("\n❌ File %s .tar not valid!\n", f)
		} else {
			fmt.Printf("\n❌ File %s .tar not valid! Error: %s\n", f, err)
		}

		return err
	}

	if err := extractor.Extract(f, ops); err != nil {
		fmt.Printf("⚠️ Last error processing %s: %s\n", f, err)

		return err
	}

	return nil
}
