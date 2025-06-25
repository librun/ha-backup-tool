package tarextractor

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/librun/ha-backup-tool/internal/options"
)

const (
	UnpackDirMod = 0755
	ExtTar       = ".tar"
	ExtTarGz     = ".tar.gz"
)

type Extractor struct {
	ops options.CmdExtractOptions
	o   string
	r   *tar.Reader
	hl  map[string]string
	fl  []string
	fs  []string
}

func New(outputDir string, ops *options.CmdExtractOptions) *Extractor {
	return &Extractor{ops: *ops, o: outputDir}
}

func (e *Extractor) Run(r io.Reader) ([]string, []string, error) {
	e.r = tar.NewReader(r)
	e.hl = map[string]string{}
	e.fl = make([]string, 0)
	e.fs = make([]string, 0)

	if _, errS := os.Stat(e.o); os.IsNotExist(errS) {
		if err := os.Mkdir(e.o, UnpackDirMod); err != nil {
			return nil, nil, err
		}
	}

	for {
		header, err := e.r.Next()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, nil, err
		}

		if !e.checkIncludeOrExcludeFile(header.Name) {
			continue
		}

		p, errS := SanitizeArchivePath(e.o, header.Name)
		if errS != nil {
			return nil, nil, errS
		}

		if p == e.o {
			continue
		}

		if err = e.extractTarItem(header, p); err != nil {
			return nil, nil, err
		}

		e.fl = append(e.fl, p)
	}

	// create hard link after all extract
	if err := e.createHardLinks(); err != nil {
		return nil, nil, err
	}

	return e.fl, e.fs, nil
}

func (e *Extractor) extractTarItem(header *tar.Header, fp string) error {
	var err error
	switch header.Typeflag {
	case tar.TypeDir:
		err = os.Mkdir(fp, UnpackDirMod)
	case tar.TypeReg:
		err = copyFile(fp, e.r, &e.ops)
	case tar.TypeLink:
		if !e.ops.SkipCreateLinks {
			e.hl[fp] = header.Linkname
		}
	case tar.TypeSymlink:
		if !e.ops.SkipCreateLinks {
			err = os.Symlink(header.Linkname, fp)
		}
	default:
		if e.ops.Verbose {
			fmt.Printf("⚠️ ExtractTarGz: uknown type: %s in %s\n", string(header.Typeflag), header.Name)
		}

		e.fs = append(e.fs, fp)
	}

	if err != nil {
		return err
	}

	return nil
}

func (e *Extractor) createHardLinks() error {
	if e.ops.SkipCreateLinks {
		return nil
	}

	for n, o := range e.hl {
		p, err := SanitizeArchivePath(e.o, o)
		if err != nil {
			return err
		}

		if _, err = os.Stat(p); os.IsNotExist(err) {
			e.fs = append(e.fs, n)

			continue
		}

		if err = os.Link(p, n); err != nil {
			return err
		}
	}

	return nil
}

func (e *Extractor) checkIncludeOrExcludeFile(fileName string) bool {
	// if not include all
	if e.ops.Include != nil {
		fi := false
		for _, i := range e.ops.Include {
			if i.MatchString(fileName) {
				fi = true

				break
			}
		}

		if !fi {
			return false
		}
	}

	fe := false
	for _, e := range e.ops.Exclude {
		if e.MatchString(fileName) {
			fe = true

			break
		}
	}

	return !fe
}
