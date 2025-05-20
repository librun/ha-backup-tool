package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/librun/ha-backup-tool/decryptor"
)

const (
	extTar          = ".tar"
	extTarGz        = ".tar.gz"
	unpackDirMod    = 0755
	regexpKeyFormat = "([A-Z0-9]{4}-){6}[A-Z0-9]{4}"
)

var (
	regexpKeyValidate = regexp.MustCompile("^" + regexpKeyFormat + "$")
	regexpKeyExtract  = regexp.MustCompile(regexpKeyFormat)

	errFileNotValid            = errors.New("file not valid")
	errEmergencyFileNotHaveKey = errors.New("emergency file not have key")
	errPasswordNotValid        = errors.New("password not valid format")
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
	key, err := getKey(c.String("e"), c.String("p"))
	if err != nil {
		return err
	}

	file := c.String("b")
	if err = validateTarFile(file); err != nil {
		fmt.Println("❌ Error: No .tar valid!")

		return err
	}

	fmt.Printf("📁 Found %s backup file(s) to process\n", file)

	successCount, err := extract(file, key, c.String("o"))
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

func getKey(e, p string) (string, error) {
	var key string

	switch {
	case e != "":
		t, err := extractKeyFromKit(e)
		if err != nil {
			fmt.Println("⚠️  Could not find encryption key in emergency kit file.")

			return "", err
		}

		t = strings.TrimSpace(t)

		key = t

		fmt.Println("✅ Found encryption key in " + key)
	case p != "":
		p = strings.TrimSpace(p)

		if !keyValidate(p) {
			fmt.Println("❌ Invalid key format.")

			return "", errPasswordNotValid
		}

		key = p
		fmt.Println("✅ Key format verified")
	default:
		fmt.Println("\nPlease enter your encryption key manually.")
		fmt.Println("It should be in the format: XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX")

		for {
			t, err := getKetManual()
			if err != nil {
				return "", err
			}

			t = strings.TrimSpace(t)

			if !keyValidate(t) {
				fmt.Println("❌ Invalid key format. Please try again.")

				continue
			}

			key = t
			fmt.Println("✅ Key format verified")

			break
		}
	}

	return key, nil
}

func getKetManual() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return text, nil
}

func keyValidate(k string) bool {
	return regexpKeyValidate.MatchString(k)
}

func extractKeyFromKit(p string) (string, error) {
	s, err := os.Stat(p)
	if err != nil {
		return "", err
	}

	if s.IsDir() {
		return "", errFileNotValid
	}

	br, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}

	b := regexpKeyExtract.Find(br)

	if b == nil {
		return "", errEmergencyFileNotHaveKey
	}

	return string(b), nil
}

func validateTarFile(p string) error {
	s, err := os.Stat(p)
	if err != nil {
		return err
	}

	if s.IsDir() {
		return errFileNotValid
	}

	ext := filepath.Ext(s.Name())
	if strings.ToLower(ext) != extTar {
		return errFileNotValid
	}

	return nil
}

func extract(file, key, outputDir string) (int, error) {
	var successCount int

	fmt.Printf("📦 Extracting %s...\n", file)
	d, err := extractBackup(file, outputDir)
	if err != nil {
		return 0, err
	}

	// Look for encrypted tar.gz files in the extracted directory
	sts := filterTarGz(d)
	if len(sts) == 0 {
		return 0, nil
	}

	var lastErr error
	for _, st := range sts {
		fn := filepath.Base(st)
		fmt.Printf("🔓 Decrypting %s...\n", fn)
		if err = extractSecureTar(st, key); err != nil {
			lastErr = err

			continue
		}

		// Remove the encrypted file after successful extraction
		if err = os.Remove(st); err != nil {
			lastErr = err

			continue
		}

		successCount++
	}

	return successCount, lastErr
}

func extractBackup(file, outputDir string) ([]string, error) {
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
	if dir == "" {
		fn := filepath.Base(file)
		fn, _ = strings.CutSuffix(fn, extTarGz)
		fn, _ = strings.CutSuffix(fn, extTar)
		dir = filepath.Join(filepath.Dir(file), fn)
	}

	if _, errS := os.Stat(dir); os.IsNotExist(errS) {
		if err = os.Mkdir(dir, unpackDirMod); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("dir %s is exists", dir) //nolint:err113 // Dynamic error
	}

	return extractTar(r, dir)
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

		p := filepath.Join(outputDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.Mkdir(p, unpackDirMod); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			outFile, errO := os.Create(p)
			if errO != nil {
				return nil, errO
			}
			if _, err = io.Copy(outFile, tarReader); err != nil {
				return nil, err
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

func extractSecureTar(file, passwd string) error {
	key, err := decryptor.PasswordToKey(passwd)
	if err != nil {
		return err
	}

	f, errO := os.Open(file)
	if errO != nil {
		return errO
	}
	defer func() {
		if err = f.Close(); err != nil {
			panic(err)
		}
	}()

	r, err := decryptor.NewReader(f, key)
	if err != nil {
		return err
	}

	if err = extractTarGz(r, file, ""); err != nil {
		fmt.Println("❌ Error: Unable to extract SecureTar - possible wrong password or file is not encrypted")

		return err
	}

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
		fn := filepath.Base(filename)
		fn, _ = strings.CutSuffix(fn, extTarGz)
		dir = filepath.Join(filepath.Dir(filename), fn)
	}

	_, err = extractTar(rg, dir)

	return err
}
