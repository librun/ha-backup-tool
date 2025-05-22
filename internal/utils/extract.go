package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/librun/ha-backup-tool/internal/decryptor"
)

const (
	extTar       = ".tar"
	extTarGz     = ".tar.gz"
	UnpackDirMod = 0755

	maxDecompressionSize int64 = 2 * 1025 * 1024 * 1024
)

var (
	ErrMaxDecompressionSize = fmt.Errorf("size of decoded data exceeds allowed size %d", maxDecompressionSize)
	ErrFileNotValid         = errors.New("file not valid")
	ErrNotFullUnpack        = errors.New("one or more files not success unpack")
)

// Extract start unpack archive.
func Extract(file, key, outputDir string, includeBackupName bool) error {
	var successCount int

	fmt.Printf("📦 Extracting %s...\n", file)
	d, err := ExtractBackup(file, outputDir, includeBackupName)
	if err != nil {
		return err
	}

	// Look for tar.gz files in the extracted directory
	sts := filterTarGz(d)
	if len(sts) == 0 {
		return nil
	}

	var lastErr error

	wg := sync.WaitGroup{}
	for _, st := range sts {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if errD := decryptArchive(file, st, key); errD != nil {
				lastErr = errD

				return
			}

			successCount++
		}()
	}

	wg.Wait()

	return lastErr
}

// ExtractBackup - unpack base tar file.
func ExtractBackup(file, outputDir string, includeBackupName bool) ([]string, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = r.Close(); err != nil {
			panic(err)
		}
	}()

	dir := outputDir

	if includeBackupName && dir != "" {
		dir = filepath.Join(dir, getBaseNameArchive(file))
	} else if dir == "" {
		dir = filepath.Join(filepath.Dir(file), getBaseNameArchive(file))
	}

	if _, errS := os.Stat(dir); os.IsNotExist(errS) {
		if err = os.Mkdir(dir, UnpackDirMod); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("dir %s is exists", dir) //nolint:err113 // Dynamic error
	}

	return extractTar(r, dir)
}

func ValidateTarFile(p string) error {
	s, err := os.Stat(p)
	if err != nil {
		return err
	}

	if s.IsDir() {
		return ErrFileNotValid
	}

	ext := filepath.Ext(s.Name())
	if strings.ToLower(ext) != extTar {
		return ErrFileNotValid
	}

	return nil
}

func extractTar(r io.Reader, outputDir string) ([]string, error) {
	tarReader := tar.NewReader(r)

	var fl []string

	for {
		header, err := tarReader.Next()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		p, errS := sanitizeArchivePath(outputDir, header.Name)
		if errS != nil {
			return nil, errS
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.Mkdir(p, UnpackDirMod); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			outFile, errO := os.Create(p)
			if errO != nil {
				return nil, errO
			}
			written, errW := io.CopyN(outFile, tarReader, maxDecompressionSize)
			if errW != nil && !errors.Is(errW, io.EOF) {
				return nil, errW
			} else if written == maxDecompressionSize {
				return nil, ErrMaxDecompressionSize
			}
			outFile.Close()

		default:
			//nolint:err113 // Dynamic error
			return nil, fmt.Errorf("ExtractTarGz: uknown type: %b in %s", header.Typeflag, header.Name)
		}

		fl = append(fl, p)
	}

	return fl, nil
}

func filterTarGz(fl []string) []string {
	var fltg []string

	for _, f := range fl {
		bn := filepath.Base(f)

		if strings.HasSuffix(strings.ToLower(bn), extTarGz) {
			fltg = append(fltg, f)
		}
	}

	return fltg
}

func decryptArchive(archName, fpath, key string) error {
	if err := extractSecureTar(archName, fpath, key); err != nil {
		return err
	}

	// Remove the encrypted file after successful extraction
	if err := os.Remove(fpath); err != nil {
		return err
	}

	return nil
}

func extractSecureTar(archName, filename, passwd string) error {
	fn := filepath.Base(filename)

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			panic(err)
		}
	}()

	r, err := decryptor.NewReader(f, passwd)
	if err != nil {
		return err
	}

	if err = extractTarGz(r, filename, ""); err != nil {
		fmt.Printf("❌ Error: Unable to extract %s/%s - possible wrong password or broken file\n", archName, fn)

		return err
	}

	fmt.Printf("🔓 Decrypt success %s/%s... \n", archName, fn)

	return nil
}

// extractTarGz - unpack tar.gz files after encrypt
func extractTarGz(r io.Reader, filename, outputDir string) error {
	rg, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	dir := outputDir
	if dir == "" {
		dir = filepath.Join(filepath.Dir(filename), getBaseNameArchive(filename))
	}

	_, err = extractTar(rg, dir)

	return err
}

// sanitize archive file pathing from "G305: Zip Slip vulnerability"
func sanitizeArchivePath(d, t string) (string, error) {
	v := filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	//nolint:err113 // Dynamic error
	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}

func getBaseNameArchive(fpath string) string {
	fn := filepath.Base(fpath)
	fn, _ = strings.CutSuffix(fn, extTarGz)
	fn, _ = strings.CutSuffix(fn, extTar)

	return fn
}
