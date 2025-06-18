package options

import (
	"regexp"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/librun/ha-backup-tool/internal/datasize"
	"github.com/librun/ha-backup-tool/internal/key"
)

const (
	maxDecompressionSize int64 = 500 * int64(datasize.GigabyteSize) // 500GB
	BackupJSON                 = "backup.json"

	
)

type GlobalOptions struct {
	Key            *key.Storage
	Verbose        bool
	MaxArchiveSize int64
}

type CmdExtractOptions struct {
	GlobalOptions
	Include         []*regexp.Regexp
	Exclude         []*regexp.Regexp
	OutputDir       string
	ExtractToSubDir bool
	SkipCreateLinks bool
}

func NewOptionFromGlobalFlags(c *cli.Command) (*GlobalOptions, error) {
	var op GlobalOptions

	var e = c.String("emergency")
	var p = c.String("password")

	op.Key = key.NewStorage(e, p)

	if c.Bool("verbose") {
		op.Verbose = true
	}

	op.MaxArchiveSize = maxDecompressionSize
	var msa = c.String("max-archive-size")
	if msa != "" {
		s, err := datasize.ParseDataSize(msa)
		if err != nil {
			return nil, err
		}

		op.MaxArchiveSize = s
	}

	return &op, nil
}

func NewCmdExtractOptions(c *cli.Command) (*CmdExtractOptions, error) {
	opg, err := NewOptionFromGlobalFlags(c)
	if err != nil {
		return nil, err
	}

	var op = CmdExtractOptions{GlobalOptions: *opg}

	op.OutputDir = c.String("output")
	op.Include, op.Exclude = parseIncudeExclude(c.String("include"), c.String("exclude"))
	op.SkipCreateLinks = c.Bool("skip-links")

	return &op, nil
}

func parseIncudeExclude(include, exclude string) ([]*regexp.Regexp, []*regexp.Regexp) {
	var ic []*regexp.Regexp
	var ec []*regexp.Regexp

	if include != "" {
		for i := range strings.SplitSeq(include, ",") {
			r := strings.ReplaceAll(i, "*", ".*")
			ic = append(ic, regexp.MustCompile("^"+r+"$"))
		}
		ic = append(ic, regexp.MustCompile("^.*"+BackupJSON+"$"))
	}

	if exclude != "" {
		for e := range strings.SplitSeq(exclude, ",") {
			r := strings.ReplaceAll(e, "*", ".*")
			ec = append(ec, regexp.MustCompile("^"+r+"$"))
		}
	}

	return ic, ec
}
