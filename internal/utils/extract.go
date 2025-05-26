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
	"regexp"
	"strings"
	"sync"

	"github.com/librun/ha-backup-tool/internal/decryptor"
)

const (
	UnpackDirMod = 0755
	extTar       = ".tar"
	extTarGz     = ".tar.gz"
	backupJSON   = "backup.json"

	maxDecompressionSize int64 = 2 * 1025 * 1024 * 1024
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
	ErrMaxDecompressionSize = fmt.Errorf("size of decoded data exceeds allowed size %d", maxDecompressionSize)
	ErrFileNotValid         = errors.New("file not valid")
	ErrNotFullUnpack        = errors.New("one or more files not success unpack")
	ErrBackupJSONNotHave    = fmt.Errorf("file %s not have", backupJSON)
	ErrBackupJSONUnmarshal  = fmt.Errorf("error unmarshal %s file", backupJSON)
	ErrBackupJSONValidate   = fmt.Errorf("error validate %s file", backupJSON)
)

// Extract - start unpack archive.
func Extract(file, outputDir, include, exclude string, key *KeyStorage, includeBackupName bool) error {
	var successCount int

	fmt.Printf("📦 Extracting %s...\n", file)
	d, err := ExtractBackup(file, outputDir, include, exclude, includeBackupName)
	if err != nil {
		return err
	}

	e, err := getBackupJSON(file, key, d)
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

			if errD := ExtractBackupItem(file, st, key, e.Protected); errD != nil {
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
func ExtractBackup(file, outputDir, include, exclude string, includeBackupName bool) ([]string, error) {
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

	if _, errS := os.Stat(dir); errS == nil {
		return nil, fmt.Errorf("dir %s is exists", dir) //nolint:err113 // Dynamic error
	}

	ic, ec := parseIncudeExclude(include, exclude)

	return extractTar(r, dir, ic, ec)
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
func ExtractBackupItem(archName, fpath string, key *KeyStorage, protected bool) error {
	fn := filepath.Base(fpath)

	var k string
	if protected {
		var err error
		k, err = key.GetKey()
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

	if err = extractTarGz(r, fpath, ""); err != nil {
		fmt.Printf("❌ Error: Unable to extract %s/%s - possible wrong password or broken file\n", archName, fn)

		return err
	}

	fmt.Printf("🔓 Extract success %s/%s... \n", archName, fn)

	// Remove the file after successful extraction
	if err = os.Remove(fpath); err != nil {
		return err
	}

	return nil
}

func extractTar(r io.Reader, outputDir string, include, exclude []*regexp.Regexp) ([]string, error) {
	tarReader := tar.NewReader(r)

	var fl []string

	if _, errS := os.Stat(outputDir); os.IsNotExist(errS) {
		if err := os.Mkdir(outputDir, UnpackDirMod); err != nil {
			return nil, err
		}
	}

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

		if p == outputDir || !checkIncludeOrExcludeFile(header.Name, include, exclude) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.Mkdir(p, UnpackDirMod)
		case tar.TypeReg:
			err = copyFile(p, tarReader)
		default:
			//nolint:err113 // Dynamic error
			err = fmt.Errorf("ExtractTarGz: uknown type: %b in %s", header.Typeflag, header.Name)
		}

		if err != nil {
			return nil, err
		}

		fl = append(fl, p)
	}

	return fl, nil
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

func getBackupJSON(file string, key *KeyStorage, fl []string) (*HomeAssistantBackup, error) {
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

		if bn == backupJSON {
			h = true
			if err := openAndUnmarshalJSON(f, &e); err != nil {
				fmt.Printf("❌ Backup %s error unmarshal %s: %s\n", file, backupJSON, err)

				return nil, ErrBackupJSONUnmarshal
			}
		}
	}

	if !h {
		fmt.Printf("⚠️  Backup %s not have %s\n", file, backupJSON)

		e = &HomeAssistantBackup{Compressed: hgz}

		if key.IsEmKitPathSet() || key.IsPasswordSet() {
			e.Protected = true
		}

		return e, nil
	}

	if err := validateBackupJSON(e); err != nil {
		fmt.Printf("❌ Backup %s error validate %s: %s\n", file, backupJSON, err)

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
func extractTarGz(r io.Reader, filename, outputDir string) error {
	rg, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	dir := outputDir
	if dir == "" {
		dir = filepath.Join(filepath.Dir(filename), getBaseNameArchive(filename))
	}

	_, err = extractTar(rg, dir, nil, nil)

	return err
}

func copyFile(fpath string, r io.Reader) error {
	outFile, err := os.Create(fpath)
	if err != nil {
		return err
	}

	defer outFile.Close()

	written, errW := io.CopyN(outFile, r, maxDecompressionSize)
	if errW != nil && !errors.Is(errW, io.EOF) {
		return errW
	} else if written == maxDecompressionSize {
		return ErrMaxDecompressionSize
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

func parseIncudeExclude(include, exclude string) ([]*regexp.Regexp, []*regexp.Regexp) {
	var ic []*regexp.Regexp
	var ec []*regexp.Regexp

	if include != "" {
		for i := range strings.SplitSeq(include, ",") {
			r := strings.ReplaceAll(i, "*", ".*")
			ic = append(ic, regexp.MustCompile("^"+r+"$"))
		}
		ic = append(ic, regexp.MustCompile("^.*"+backupJSON+"$"))
	}

	if exclude != "" {
		for e := range strings.SplitSeq(exclude, ",") {
			r := strings.ReplaceAll(e, "*", ".*")
			ec = append(ec, regexp.MustCompile("^"+r+"$"))
		}
	}

	return ic, ec
}

func checkIncludeOrExcludeFile(fileName string, include, exclude []*regexp.Regexp) bool {
	// if not include all
	if include != nil {
		fi := false
		for _, i := range include {
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
	for _, e := range exclude {
		if e.MatchString(fileName) {
			fe = true

			break
		}
	}

	return !fe
}
