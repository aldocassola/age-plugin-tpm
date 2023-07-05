package plugin

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/foxboron/age-plugin-tpm/internal/bech32"
	"github.com/google/go-tpm/tpmutil"
)

// We need to know if the TPM handle has a pin set
type PINStatus uint8

const (
	NoPIN PINStatus = iota
	HasPIN
)

func (p PINStatus) String() string {
	switch p {
	case NoPIN:
		return "NoPIN"
	case HasPIN:
		return "HasPIN"
	}
	return "Not a PINStatus"
}

type Identity struct {
	Version uint8
	PIN     PINStatus
	Private tpmutil.U16Bytes
	Public  tpmutil.U16Bytes
}

func (i *Identity) Serialize() []any {
	return []interface{}{
		&i.Version,
		&i.PIN,
	}
}

func DecodeIdentity(s string) (*Identity, error) {
	var key Identity
	hrp, b, err := bech32.Decode(s)
	if err != nil {
		return nil, err
	}
	if hrp != strings.ToUpper(IdentityPrefix) {
		return nil, fmt.Errorf("invalid hrp")
	}
	r := bytes.NewBuffer(b)
	for _, f := range key.Serialize() {
		if err := binary.Read(r, binary.BigEndian, f); err != nil {
			return nil, err
		}
	}

	if err := tpmutil.UnpackBuf(r, &key.Private, &key.Public); err != nil {
		return nil, fmt.Errorf("failed unpacking buffer: %v", err)
	}

	return &key, nil
}

func ParseIdentity(f io.Reader) (*Identity, error) {
	// Same parser as age
	const privateKeySizeLimit = 1 << 24 // 16 MiB
	scanner := bufio.NewScanner(io.LimitReader(f, privateKeySizeLimit))
	var n int
	for scanner.Scan() {
		n++
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		identity, err := DecodeIdentity(line)
		if err != nil {
			return nil, fmt.Errorf("error at line %d: %v", n, err)
		}
		return identity, nil
	}
	return nil, fmt.Errorf("no identites found")
}

func EncodeIdentity(i *Identity) (string, error) {
	var b bytes.Buffer
	for _, v := range i.Serialize() {
		if err := binary.Write(&b, binary.BigEndian, v); err != nil {
			return "", err
		}
	}

	packed, err := tpmutil.Pack(&i.Private, &i.Public)
	if err != nil {
		return "", fmt.Errorf("failed packing TPM key: %v", err)
	}
	b.Write(packed)
	s, err := bech32.Encode(strings.ToUpper(IdentityPrefix), b.Bytes())
	if err != nil {
		return "", err
	}
	return s, nil
}

var (
	marshalTemplate = `
# Created: %s
`
)

func Marshal(i *Identity, w io.Writer) {
	s := fmt.Sprintf(marshalTemplate, time.Now())
	s = strings.TrimSpace(s)
	fmt.Fprintf(w, "%s\n", s)
}

func MarshalIdentity(i *Identity, recipient string, w io.Writer) error {
	key, err := EncodeIdentity(i)
	if err != nil {
		return err
	}
	Marshal(i, w)
	fmt.Fprintf(w, "# Recipient: %s\n", recipient)
	fmt.Fprintf(w, "\n%s\n", key)
	return nil
}
