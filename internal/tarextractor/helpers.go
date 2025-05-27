package tarextractor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/librun/ha-backup-tool/internal/options"
)

// Sanitize archive file pathing from "G305: Zip Slip vulnerability"
func SanitizeArchivePath(d, t string) (string, error) {
	v := filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	//nolint:err113 // Dynamic error
	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}

// GetBaseNameArchive - get base archive name without ext and location.
func GetBaseNameArchive(fpath string) string {
	fn := filepath.Base(fpath)
	fn, _ = strings.CutSuffix(fn, ExtTarGz)
	fn, _ = strings.CutSuffix(fn, ExtTar)

	return fn
}

func copyFile(fpath string, r io.Reader, ops *options.CmdExtractOptions) error {
	outFile, err := os.Create(fpath)
	if err != nil {
		return err
	}

	defer outFile.Close()

	written, errW := io.CopyN(outFile, r, ops.MaxArchiveSize)
	if errW != nil && !errors.Is(errW, io.EOF) {
		return errW
	} else if written == ops.MaxArchiveSize {
		return fmt.Errorf("size of decoded data exceeds allowed size %d", ops.MaxArchiveSize) //nolint:err113 // Dynamic error
	}

	return nil
}
