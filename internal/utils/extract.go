package utils

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/librun/ha-backup-tool/internal/decryptor"
	"github.com/librun/ha-backup-tool/internal/logger"
	"github.com/librun/ha-backup-tool/internal/options"
	"github.com/librun/ha-backup-tool/internal/tarextractor"
)

type tarGzReader struct {
	io.Reader
	file *os.File
}

//nolint:gochecknoglobals // This is const varible
var (
	backupJSONCryptSupport   = []string{"", "aes128"}
	backupJSONVersionSupport = []int{2}
)

var (
	ErrFileNotValid        = errors.New("file not valid")
	ErrNotFullUnpack       = errors.New("one or more files not success unpack")
	ErrBackupJSONNotHave   = fmt.Errorf("file %s not have", options.BackupJSON)
	ErrBackupJSONUnmarshal = fmt.Errorf("error unmarshal %s file", options.BackupJSON)
	ErrBackupJSONValidate  = fmt.Errorf("error validate %s file", options.BackupJSON)
)

// Extract - start unpack archive.
func Extract(file string, ops *options.CmdExtractOptions) error {
	var successCount int

	fmt.Printf("📦 Extracting %s...\n", file)
	d, err := ExtractBackup(file, ops)
	if err != nil {
		return err
	}

	e, err := getBackupJSON(file, d, ops)
	if err != nil {
		return err
	}

	if !e.Compressed {
		return nil
	}

	// Look for tar.gz files in the extracted directory
	sts := filterFilesBySuffix(d, tarextractor.ExtTarGz)
	if len(sts) == 0 {
		return nil
	}

	var lastErr error

	wg := sync.WaitGroup{}
	for _, st := range sts {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if errE := ExtractBackupItem(file, st, e.Protected, ops); errE != nil {
				if ops.Verbose {
					fmt.Printf("❌ Failed extract from backup: %s/%s encrypted: %t Error: %s\n",
						file, filepath.Base(st), e.Protected, errE)
				}

				lastErr = errE

				return
			}

			// Remove the file after successful extraction
			if errR := os.Remove(st); errR != nil {
				if ops.Verbose {
					fmt.Printf("❌ Failed delete file: %s/%s Error: %s\n", file, filepath.Base(st), errR)
				}

				lastErr = errR

				return
			}

			successCount++
		}()
	}

	wg.Wait()

	return lastErr
}

// ExtractBackup - unpack base tar file.
func ExtractBackup(file string, ops *options.CmdExtractOptions) ([]string, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = r.Close(); err != nil {
			logger.Fatal("Backup: %s Error close file: %s", file, err)
		}
	}()

	dir := ops.OutputDir

	if ops.ExtractToSubDir && dir != "" {
		dir = filepath.Join(dir, tarextractor.GetBaseNameArchive(file))
	} else if dir == "" {
		dir = filepath.Join(filepath.Dir(file), tarextractor.GetBaseNameArchive(file))
	}

	if _, errS := os.Stat(dir); errS == nil {
		return nil, fmt.Errorf("dir %s is exists", dir) //nolint:err113 // Dynamic error
	}

	te := tarextractor.New(dir, ops)
	fl, fs, errE := te.Run(r)
	if len(fs) > 0 {
		fmt.Printf("⚠️ In progress extract %s skipped %d file(s)\n", file, len(fs))
	}

	return fl, errE
}

// ValidateTarFile - validate tar file is exist and other.
func ValidateTarFile(p string) error {
	s, err := os.Stat(p)
	if err != nil {
		return err
	}

	if s.IsDir() {
		return ErrFileNotValid
	}

	ext := filepath.Ext(s.Name())
	if strings.ToLower(ext) != tarextractor.ExtTar {
		return ErrFileNotValid
	}

	return nil
}

// ExtractBackupItem - function for extract backup sub archive.
func ExtractBackupItem(archName, fpath string, protected bool, ops *options.CmdExtractOptions) error {
	fn := filepath.Base(fpath)

	var k string
	if protected {
		var err error
		k, err = ops.Key.GetKey()
		if err != nil {
			return err
		}
	}

	r, err := newTarGzReader(fpath, k, protected)
	if err != nil {
		return err
	}
	defer func() {
		if err = r.Close(); err != nil {
			logger.Fatal("File %s/%s Error close file: %s", archName, fn, err)
		}
	}()

	if err = extractTarGz(r, fpath, "", ops); err != nil {
		fmt.Printf("❌ Unable to extract %s/%s - possible wrong password or broken file\n", archName, fn)

		return err
	}

	fmt.Printf("🔓 Extract success %s/%s... \n", archName, fn)

	return nil
}

func filterFilesBySuffix(fl []string, suffix string) []string {
	var fltg []string

	suffix = strings.ToLower(suffix)

	for _, f := range fl {
		bn := filepath.Base(f)

		if strings.HasSuffix(strings.ToLower(bn), suffix) {
			fltg = append(fltg, f)
		}
	}

	return fltg
}

func getBackupJSON(file string, fl []string, ops *options.CmdExtractOptions) (*HomeAssistantBackup, error) {
	var e *HomeAssistantBackup
	var h bool
	var hgz bool

	for _, f := range fl {
		bn := filepath.Base(f)
		bn = strings.ToLower(bn)

		if strings.HasSuffix(bn, tarextractor.ExtTarGz) {
			hgz = true

			continue
		}

		if bn == options.BackupJSON {
			h = true
			if err := openAndUnmarshalJSON(f, &e); err != nil {
				fmt.Printf("❌ Backup %s error unmarshal %s: %s\n", file, options.BackupJSON, err)

				return nil, ErrBackupJSONUnmarshal
			}
		}
	}

	if !h {
		if ops.Verbose {
			fmt.Printf("⚠️ Backup %s not have %s\n", file, options.BackupJSON)
		}

		e = &HomeAssistantBackup{Compressed: hgz}

		if ops.Key.IsEmKitPathSet() || ops.Key.IsPasswordSet() {
			e.Protected = true
		}

		return e, nil
	}

	if err := validateBackupJSON(e); err != nil {
		fmt.Printf("❌ Backup %s error validate %s: %s\n", file, options.BackupJSON, err)

		return nil, ErrBackupJSONValidate
	}

	return e, nil
}

func openAndUnmarshalJSON(fpath string, v any) error {
	fo, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer func() {
		if err = fo.Close(); err != nil {
			logger.Fatal("File: %s Error close file: %s", fpath, err)
		}
	}()

	b, err := io.ReadAll(fo)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(b, v); err != nil {
		return err
	}

	return nil
}

func validateBackupJSON(e *HomeAssistantBackup) error {
	var c = strings.ToLower(e.Crypto)
	var cs bool
	var vs bool

	for _, s := range backupJSONCryptSupport {
		if s == c {
			cs = true
		}
	}

	if !cs {
		return fmt.Errorf("crypto type %s not support", e.Crypto) //nolint:err113 // Dynamic error
	}

	for _, s := range backupJSONVersionSupport {
		if s == e.Version {
			vs = true
		}
	}

	if !vs {
		return fmt.Errorf("version backup %d not support", e.Version) //nolint:err113 // Dynamic error
	}

	return nil
}

func newTarGzReader(filename, passwd string, protected bool) (*tarGzReader, error) {
	var re tarGzReader

	var err error
	re.file, err = os.Open(filename)
	if err != nil {
		return nil, err
	}

	if !protected {
		re.Reader = re.file

		return &re, nil
	}

	re.Reader, err = decryptor.NewReader(re.file, passwd)
	if err != nil {
		return nil, err
	}

	return &re, nil
}

func (r *tarGzReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}

	return nil
}

// extractTarGz - unpack tar.gz files after encrypt
func extractTarGz(r io.Reader, filename, outputDir string, ops *options.CmdExtractOptions) error {
	rg, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	dir := outputDir
	if dir == "" {
		dir = filepath.Join(filepath.Dir(filename), tarextractor.GetBaseNameArchive(filename))
	}

	te := tarextractor.New(dir, ops)
	_, fs, errE := te.Run(rg)
	if len(fs) > 0 {
		bn := filepath.Base(filename)
		fmt.Printf("⚠️ In progress extract %s skipped %d file(s)\n", bn, len(fs))
	}

	return errE
}
