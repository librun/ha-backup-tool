package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
	// "github.com/urfave/cli-docs/v3"

	"github.com/librun/ha-backup-tool/internal/commands"
)

// AppVersion displays service version in semantic versioning (http://semver.org/).
// Can be replaced while compiling with flag "-ldflags "-X main.AppVersion=${VERSION}"".
var AppVersion = "develop" //nolint:gochecknoglobals // Use for replace outside

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
			&cli.StringFlag{
				Name:  "max-archive-size",
				Usage: "Max size for extract archive",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Verbose mode for output more information",
			},
		},
		Commands: []*cli.Command{
			commands.Extract(),
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
