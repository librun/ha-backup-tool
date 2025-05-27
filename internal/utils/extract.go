package utils

import (
	"archive/tar"
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
	"github.com/librun/ha-backup-tool/internal/options"
)

const (
	UnpackDirMod = 0755
	extTar       = ".tar"
	extTarGz     = ".tar.gz"
)

type tarGzReader struct {
	io.Reader
	file *os.File
}

//nolint:gochecknoglobals // This is const varible
var (
	backupJSONCryptSupport   = []string{"aes128"}
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
	sts := filterFilesBySuffix(d, extTarGz)
	if len(sts) == 0 {
		return nil
	}

	var lastErr error

	wg := sync.WaitGroup{}
	for _, st := range sts {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if errD := ExtractBackupItem(file, st, e.Protected, ops); errD != nil {
				if ops.Verbose {
					fmt.Printf("❌ Failed extract from backup: %s/%s encrypted: %t Error: %s\n",
						file, filepath.Base(st), e.Protected, errD)
				}

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
func ExtractBackup(file string, ops *options.CmdExtractOptions) ([]string, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = r.Close(); err != nil {
			panic(err)
		}
	}()

	dir := ops.OutputDir

	if ops.ExtractToSubDir && dir != "" {
		dir = filepath.Join(dir, getBaseNameArchive(file))
	} else if dir == "" {
		dir = filepath.Join(filepath.Dir(file), getBaseNameArchive(file))
	}

	if _, errS := os.Stat(dir); errS == nil {
		return nil, fmt.Errorf("dir %s is exists", dir) //nolint:err113 // Dynamic error
	}

	fl, fs, errE := extractTar(r, dir, ops)
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
	if strings.ToLower(ext) != extTar {
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
		if r.file != nil {
			if err = r.file.Close(); err != nil {
				panic(err)
			}
		}
	}()

	if err = extractTarGz(r, fpath, "", ops); err != nil {
		fmt.Printf("❌ Unable to extract %s/%s - possible wrong password or broken file\n", archName, fn)

		return err
	}

	fmt.Printf("🔓 Extract success %s/%s... \n", archName, fn)

	// Remove the file after successful extraction
	if err = os.Remove(fpath); err != nil {
		return err
	}

	return nil
}

func extractTar(r io.Reader, outputDir string, ops *options.CmdExtractOptions) ([]string, []string, error) {
	tarReader := tar.NewReader(r)

	var fl []string
	var fs []string

	if _, errS := os.Stat(outputDir); os.IsNotExist(errS) {
		if err := os.Mkdir(outputDir, UnpackDirMod); err != nil {
			return nil, nil, err
		}
	}

	for {
		header, err := tarReader.Next()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, nil, err
		}

		p, errS := sanitizeArchivePath(outputDir, header.Name)
		if errS != nil {
			return nil, nil, errS
		}

		if p == outputDir || !checkIncludeOrExcludeFile(header.Name, ops) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.Mkdir(p, UnpackDirMod)
		case tar.TypeReg:
			err = copyFile(p, tarReader, ops)
		default:
			if ops.Verbose {
				fmt.Printf("⚠️ ExtractTarGz: uknown type: %s in %s\n", string(header.Typeflag), header.Name)
			}

			fs = append(fs, p)
		}

		if err != nil {
			return nil, nil, err
		}

		fl = append(fl, p)
	}

	return fl, fs, nil
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

		if strings.HasSuffix(bn, extTarGz) {
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
			panic(err)
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

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	if !protected {
		re.file = f
		re.Reader = f

		return &re, nil
	}

	re.Reader, err = decryptor.NewReader(f, passwd)
	if err != nil {
		return nil, err
	}

	return &re, nil
}

// extractTarGz - unpack tar.gz files after encrypt
func extractTarGz(r io.Reader, filename, outputDir string, ops *options.CmdExtractOptions) error {
	rg, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	dir := outputDir
	if dir == "" {
		dir = filepath.Join(filepath.Dir(filename), getBaseNameArchive(filename))
	}

	_, fs, errE := extractTar(rg, dir, ops)
	if len(fs) > 0 {
		bn := filepath.Base(filename)
		fmt.Printf("⚠️ In progress extract %s skipped %d file(s)\n", bn, len(fs))
	}

	return errE
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

// sanitize archive file pathing from "G305: Zip Slip vulnerability"
func sanitizeArchivePath(d, t string) (string, error) {
	v := filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	//nolint:err113 // Dynamic error
	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}

// getBaseNameArchive - get base archive name without ext and location.
func getBaseNameArchive(fpath string) string {
	fn := filepath.Base(fpath)
	fn, _ = strings.CutSuffix(fn, extTarGz)
	fn, _ = strings.CutSuffix(fn, extTar)

	return fn
}

func checkIncludeOrExcludeFile(fileName string, ops *options.CmdExtractOptions) bool {
	// if not include all
	if ops.Include != nil {
		fi := false
		for _, i := range ops.Include {
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
	for _, e := range ops.Exclude {
		if e.MatchString(fileName) {
			fe = true

			break
		}
	}

	return !fe
}
