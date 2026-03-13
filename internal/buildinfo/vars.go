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
	if !crypto.SHA512.Available() {
		// WASM 환경 등에서는 SHA512가 없을 수 있음
		Signature = "unavailable"
		return
	}
	hashed := crypto.SHA512.New()

	_, _ = io.WriteString(hashed, AppName)
	_, _ = io.WriteString(hashed, Version)
	_, _ = io.WriteString(hashed, BuildTime)

	Signature = hex.EncodeToString(hashed.Sum(nil))
}
