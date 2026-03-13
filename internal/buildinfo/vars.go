package buildinfo

import (
	"crypto"
	"encoding/hex"
	"io"
)

var (
	Version   string
	AppName   string
	BuildTime string

	// compute
	Signature string
)

func init() {
	hashed := crypto.SHA512.New()

	_, _ = io.WriteString(hashed, AppName)
	_, _ = io.WriteString(hashed, Version)
	_, _ = io.WriteString(hashed, BuildTime)

	Signature = hex.EncodeToString(hashed.Sum(nil))
}
