package datasize_test

import (
	"errors"
	"testing"

	"github.com/librun/ha-backup-tool/internal/datasize"
)

func TestParseDataSize(t *testing.T) {
	var td = []struct {
		Input       string
		OutputValue int64
		OutputError error
	}{
		{
			Input:       "11",
			OutputValue: 11,
		},
		{
			Input:       "12b",
			OutputValue: 12,
		},
		{
			Input:       "13B",
			OutputValue: 104,
		},
		//K
		{
			Input:       "24Kb",
			OutputValue: 24000,
		},
		{
			Input:       "25Kib",
			OutputValue: 25600,
		},
		{
			Input:       "26KB",
			OutputValue: 208000,
		},
		{
			Input:       "27KiB",
			OutputValue: 221184,
		},
		//M
		{
			Input:       "38Mb",
			OutputValue: 38000000,
		},
		{
			Input:       "39Mib",
			OutputValue: 40894464,
		},
		{
			Input:       "30MB",
			OutputValue: 240000000,
		},
		{
			Input:       "31MiB",
			OutputValue: 260046848,
		},
		//G
		{
			Input:       "22Gb",
			OutputValue: 22000000000,
		},
		{
			Input:       "23Gib",
			OutputValue: 24696061952,
		},
		{
			Input:       "24GB",
			OutputValue: 192000000000,
		},
		{
			Input:       "25GiB",
			OutputValue: 214748364800,
		},
		//T
		{
			Input:       "16Tb",
			OutputValue: 16000000000000,
		},
		{
			Input:       "17Tib",
			OutputValue: 18691697672192,
		},
		{
			Input:       "18TB",
			OutputValue: 144000000000000,
		},
		{
			Input:       "19TiB",
			OutputValue: 167125767421952,
		},
		{
			Input:       "19T",
			OutputError: datasize.ErrDatasizeNotValid,
		},
		{
			Input:       "0.0B",
			OutputError: datasize.ErrDatasizeNotValid,
		},
		{
			Input:       "0,0B",
			OutputError: datasize.ErrDatasizeNotValid,
		},
		{
			Input:       "0B",
			OutputValue: 0,
		},
		{
			Input:       "79mb",
			OutputValue: 79000000,
		},
		{
			Input:       "78mib",
			OutputValue: 81788928,
		},
		{
			Input:       "77mB",
			OutputValue: 616000000,
		},
		{
			Input:       "76miB",
			OutputValue: 637534208,
		},
		{
			Input:       "1048575TiB",
			OutputValue: 9223363240761753600,
		},
		{
			Input:       "1048576TiB",
			OutputError: datasize.ErrDatasizeNotValid,
		},
		{
			Input:       "1048576000TiB",
			OutputError: datasize.ErrDatasizeNotValid,
		},
		{
			Input:       "-10TiB",
			OutputError: datasize.ErrDatasizeNotValid,
		},
	}

	for _, v := range td {
		r, e := datasize.ParseDataSize(v.Input)

		if !errors.Is(e, v.OutputError) {
			t.Errorf("For input value %s got: %e wait %e", v.Input, e, v.OutputError)
		}

		if r != v.OutputValue {
			t.Errorf("For input value %s got: %d wait %d", v.Input, r, v.OutputValue)
		}
	}
}
