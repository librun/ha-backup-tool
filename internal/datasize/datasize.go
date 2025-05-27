package datasize

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

type Datasize int64

const (
	// base 10 (SI prefixes)
	BitSize     Datasize = 1e0
	KilobitSize          = BitSize * 1e3
	MegabitSize          = BitSize * 1e6
	GigabitSize          = BitSize * 1e9
	TerabitSize          = BitSize * 1e12

	ByteSize     = BitSize * 8
	KilobyteSize = ByteSize * 1e3
	MegabyteSize = ByteSize * 1e6
	GigabyteSize = ByteSize * 1e9
	TerabyteSize = ByteSize * 1e12

	// base 2 (IEC prefixes)
	KibibitSize = BitSize * 1024
	MebibitSize = KibibitSize * 1024
	GibibitSize = MebibitSize * 1024
	TebibitSize = GibibitSize * 1024

	KibibyteSize = ByteSize * 1024
	MebibyteSize = KibibyteSize * 1024
	GibibyteSize = MebibyteSize * 1024
	TebibyteSize = GibibyteSize * 1024

	BitLabel     = "b"
	KilobitLabel = "Kb"
	MegabitLabel = "Mb"
	GigabitLabel = "Gb"
	TerabitLabel = "Tb"

	ByteLabel     = "B"
	KilobyteLabel = "KB"
	MegabyteLabel = "MB"
	GigabyteLabel = "GB"
	TerabyteLabel = "TB"

	KibibitLabel = "Kib"
	MebibitLabel = "Mib"
	GibibitLabel = "Gib"
	TebibitLabel = "Tib"

	KibibyteLabel = "KiB"
	MebibyteLabel = "MiB"
	GibibyteLabel = "GiB"
	TebibyteLabel = "TiB"
)

const lenRegexpExtractDatasize = 3

var (
	ErrDatasizeNotValid = errors.New("datasize not valid")

	regexpExtractDatasize = regexp.MustCompile(`^(\d+)(.*)$`)
)

func ParseDataSize(v string) (int64, error) {
	mappingDataSize := map[string]Datasize{
		BitLabel:      BitSize,
		KilobitLabel:  KilobitSize,
		MegabitLabel:  MegabitSize,
		GigabitLabel:  GigabitSize,
		TerabitLabel:  TerabitSize,
		ByteLabel:     ByteSize,
		KilobyteLabel: KilobyteSize,
		MegabyteLabel: MegabyteSize,
		GigabyteLabel: GigabyteSize,
		TerabyteLabel: TerabyteSize,
		KibibitLabel:  KibibitSize,
		MebibitLabel:  MebibitSize,
		GibibitLabel:  GibibitSize,
		TebibitLabel:  TebibitSize,
		KibibyteLabel: KibibyteSize,
		MebibyteLabel: MebibyteSize,
		GibibyteLabel: GibibyteSize,
		TebibyteLabel: TebibyteSize,
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
		t = BitLabel
	case 1:
		t = d[2]
	default:
		t = strings.ToUpper(d[2][0:1]) + d[2][1:]
	}

	m, ok := mappingDataSize[t]
	if !ok {
		return 0, ErrDatasizeNotValid
	}

	if i > 1048575 && m > GibibyteSize {
		return 0, ErrDatasizeNotValid
	}

	r := i * int64(m)
	if r < 0 {
		return 0, ErrDatasizeNotValid
	}

	return r, nil
}
