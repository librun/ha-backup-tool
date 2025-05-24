package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
	// "github.com/urfave/cli-docs/v3"

	"github.com/librun/ha-backup-tool/internal/commands"
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
						Aliases: []string{"i"},
						Usage:   "Include files",
					},
					&cli.StringFlag{
						Name:    "exclude",
						Aliases: []string{"e"},
						Usage:   "Exclude files",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Directory for unpack files",
					},
				},
				Action: commands.Extract,
			},
		},
	}

	// generateDocs(app)

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// // generateDocs - function for generate docs
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
