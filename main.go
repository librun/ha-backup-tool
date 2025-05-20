package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/librun/ha-backup-tool/internal/utils"
)

const (
	AppVersion = "1.0.0"
)

func main() {
	app := &cli.Command{
		Name:                  "ha-backup-tool",
		Usage:                 "Home Assistant Tool for work with backup",
		EnableShellCompletion: true,
		Version:               AppVersion,
		Action:                runDecrypt,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "b",
				Aliases:  []string{"backup"},
				Usage:    "Filepath for backup home assistant in tar format",
			},
			&cli.StringFlag{
				Name:    "e",
				Aliases: []string{"emergency"},
				Usage:   "Filepath for emergency text file",
			},
			&cli.StringFlag{
				Name:    "p",
				Aliases: []string{"password"},
				Usage:   "Password for decrypt backup",
			},
			&cli.StringFlag{
				Name:    "o",
				Aliases: []string{"output"},
				Usage:   "Directory for unpack files",
			},
		},
	}

	// md, err := docs.ToMarkdown(app)
	// if err != nil {
	// 	panic(err)
	// }

	// fi, err := os.Create("cli-docs.md")
	// if err != nil {
	// 	panic(err)
	// }
	// defer fi.Close()
	// if _, err := fi.WriteString("# CLI\n\n" + md); err != nil {
	// 	panic(err)
	// }

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runDecrypt(_ context.Context, c *cli.Command) error {
	key, err := utils.GetKey(c.String("e"), c.String("p"))
	if err != nil {
		return err
	}

	file := c.String("b")
	if err = utils.ValidateTarFile(file); err != nil {
		fmt.Println("❌ Error: No .tar valid!")

		return err
	}

	fmt.Printf("📁 Found %s backup file(s) to process\n", file)

	successCount, err := utils.Extract(file, key, c.String("o"))
	if err != nil {
		fmt.Printf("\n❌ Error processing %s: %s\n", file, err)
	}

	if successCount > 0 {
		fmt.Printf("\n✅ Successfully decrypted %v backup file(s)!\n", successCount)
		fmt.Println("You can find the decrypted files in the extracted directories.")
	} else {
		fmt.Println("\n⚠️  No files were successfully decrypted.")
		fmt.Println("Please check that your backup files and emergency kit are correct.")
	}

	return nil
}
