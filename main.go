package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/librun/ha-backup-tool/internal/utils"
)

const (
	AppVersion = "1.1.0"
)

func main() {
	app := &cli.Command{
		Name:                  "ha-backup-tool",
		Usage:                 "Home Assistant Tool for work with backup",
		EnableShellCompletion: true,
		Version:               AppVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "emergency",
				Aliases: []string{"e"},
				Usage:   "Filepath for emergency text file",
			},
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"p"},
				Usage:   "Password for decrypt backup",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "extract",
				Aliases: []string{"e", "unpack", "u"},
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
						Name:    "o",
						Aliases: []string{"output"},
						Usage:   "Directory for unpack files",
					},
				},
				Action: cmdExtract,
			},
		},
	}

	// generateDocs(app)

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// GenerateDocs - function for generate docs
// func generateDocs(app *cli.Command) {
// 	md, err := docs.ToMarkdown(app)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fi, err := os.Create("cli-docs.md")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer fi.Close()
// 	if _, err := fi.WriteString("# CLI\n\n" + md); err != nil {
// 		panic(err)
// 	}
// }

func cmdExtract(_ context.Context, c *cli.Command) error {
	var o = c.String("output")
	var e = c.String("emergency")
	var p = c.String("password")
	var fs = c.StringArgs("backups")

	key, err := utils.GetKey(e, p)
	if err != nil {
		return err
	}

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
			if err = os.Mkdir(o, utils.UnpackDirMod); err != nil {
				return err
			}
		}
	}

	for _, f := range fs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if er := utils.ValidateTarFile(f); er != nil {
				fmt.Printf("\n❌ Error: %s. File .tar not valid!\n", er)

				return
			}

			if er := utils.Extract(f, key, o, m); er != nil {
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
