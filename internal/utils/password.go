package utils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	regexpKeyFormat = "([A-Z0-9]{4}-){6}[A-Z0-9]{4}"
)

var (
	regexpKeyValidate = regexp.MustCompile("^" + regexpKeyFormat + "$")
	regexpKeyExtract  = regexp.MustCompile(regexpKeyFormat)

	ErrEmergencyFileNotHaveKey = errors.New("emergency file not have key")
	ErrPasswordNotValid        = errors.New("password not valid format")
)

// GetKey - get password key for decrypt archive.
func GetKey(e, p string) (string, error) {
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

			return "", ErrPasswordNotValid
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
		return "", ErrFileNotValid
	}

	br, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}

	b := regexpKeyExtract.Find(br)

	if b == nil {
		return "", ErrEmergencyFileNotHaveKey
	}

	return string(b), nil
}
