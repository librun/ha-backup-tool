package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/librun/ha-backup-tool/internal/utils"
)

func main() {
	cmd := &cli.Command{
		Name:   "ha-decryptu-backup-tool",
		Usage:  "Home assistant unpack encrypt backup",
		Action: runDecrypt,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "b",
				Aliases:  []string{"backup"},
				Usage:    "Filepath for backup home assistant in tar format",
				Required: true,
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

	if err := cmd.Run(context.Background(), os.Args); err != nil {
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
