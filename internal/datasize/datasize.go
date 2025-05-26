package datasize

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

type datasize int64

const (
	// base 10 (SI prefixes)
	dBit     datasize = 1e0
	dKilobit          = dBit * 1e3
	dMegabit          = dBit * 1e6
	dGigabit          = dBit * 1e9
	dTerabit          = dBit * 1e12

	dByte     = dBit * 8
	dKilobyte = dByte * 1e3
	dMegabyte = dByte * 1e6
	dGigabyte = dByte * 1e9
	dTerabyte = dByte * 1e12

	// base 2 (IEC prefixes)
	dKibibit = dBit * 1024
	dMebibit = dKibibit * 1024
	dGibibit = dMebibit * 1024
	dTebibit = dGibibit * 1024

	dKibibyte = dByte * 1024
	dMebibyte = dKibibyte * 1024
	dGibibyte = dMebibyte * 1024
	dTebibyte = dGibibyte * 1024

	pBit     = "b"
	pKilobit = "Kb"
	pMegabit = "Mb"
	pGigabit = "Gb"
	pTerabit = "Tb"

	pByte     = "B"
	pKilobyte = "KB"
	pMegabyte = "MB"
	pGigabyte = "GB"
	pTerabyte = "TB"

	pKibibit = "Kib"
	pMebibit = "Mib"
	pGibibit = "Gib"
	pTebibit = "Tib"

	pKibibyte = "KiB"
	pMebibyte = "MiB"
	pGibibyte = "GiB"
	pTebibyte = "TiB"
)

const lenRegexpExtractDatasize = 3

var (
	ErrDatasizeNotValid = errors.New("datasize not valid")

	regexpExtractDatasize = regexp.MustCompile(`^(\d+)(.*)$`)
)

func ParseDataSize(v string) (int64, error) {
	mappingDataSize := map[string]datasize{
		pBit:      dBit,
		pKilobit:  dKilobit,
		pMegabit:  dMegabit,
		pGigabit:  dGigabit,
		pTerabit:  dTerabit,
		pByte:     dByte,
		pKilobyte: dKilobyte,
		pMegabyte: dMegabyte,
		pGigabyte: dGigabyte,
		pTerabyte: dTerabyte,
		pKibibit:  dKibibit,
		pMebibit:  dMebibit,
		pGibibit:  dGibibit,
		pTebibit:  dTebibit,
		pKibibyte: dKibibyte,
		pMebibyte: dMebibyte,
		pGibibyte: dGibibyte,
		pTebibyte: dTebibyte,
	}

	d := regexpExtractDatasize.FindStringSubmatch(v)
	if len(d) != lenRegexpExtractDatasize {
		return 0, ErrDatasizeNotValid
	}

	i, err := strconv.ParseInt(d[1], 10, 64)
	if err != nil {
		return 0, err
	}

	var t string

	switch len(d[2]) {
	case 0:
		t = pBit
	case 1:
		t = d[2]
	default:
		t = strings.ToUpper(d[2][0:1]) + d[2][1:]
	}

	m, ok := mappingDataSize[t]
	if !ok {
		return 0, ErrDatasizeNotValid
	}

	if i > 1048575 && m > dGibibyte {
		return 0, ErrDatasizeNotValid
	}

	return i * int64(m), nil
}
