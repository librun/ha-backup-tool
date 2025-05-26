package commands

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/librun/ha-backup-tool/internal/utils"
)

// Extract - command for extract archive.
func Extract() *cli.Command {
	return &cli.Command{
		Name:    "extract",
		Aliases: []string{"unpack"},
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
				Name:    "include",
				Aliases: []string{"ic"},
				Usage:   "Include files",
			},
			&cli.StringFlag{
				Name:    "exclude",
				Aliases: []string{"ec"},
				Usage:   "Exclude files",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Directory for unpack files",
			},
		},
		Action: extractAction,
	}
}

// extractAction - command for extract backups.
func extractAction(_ context.Context, c *cli.Command) error {
	var o = c.String("output")
	var e = c.String("emergency")
	var p = c.String("password")
	var fs = c.StringArgs("backups")
	var ic = c.String("include")
	var ec = c.String("exclude")

	key := utils.NewKeyStorage(e, p)

	if len(fs) == 0 {
		fmt.Println("\n⚠️  No files for extract.")

		return nil
	}

	fmt.Printf("📁 Found %s backup file(s) to process\n", fs)

	var s int
	var m = len(fs) > 1
	var wg = sync.WaitGroup{}

	if m && o != "" {
		if _, errS := os.Stat(o); os.IsNotExist(errS) {
			if err := os.Mkdir(o, utils.UnpackDirMod); err != nil {
				return err
			}
		}
	}

	for _, f := range fs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := utils.ValidateTarFile(f); err != nil {
				fmt.Printf("\n❌ Error: %s. File .tar not valid!\n", err)

				return
			}

			if er := utils.Extract(f, o, ic, ec, key, m); er != nil {
				fmt.Printf("\n❌ Error processing %s: %s\n", f, er)

				return
			}

			s++
		}()
	}

	wg.Wait()

	if s > 0 {
		fmt.Printf("\n✅ Successfully decrypted %v of %v backup file(s)!\n", s, len(fs))
		fmt.Println("You can find the decrypted files in the extracted directories.")
	} else {
		fmt.Println("\n⚠️  No files were successfully decrypted.")
	}

	if s == 0 || s != len(fs) {
		fmt.Println("Please check that your backup files and emergency kit are correct.")
	}

	return nil
}
