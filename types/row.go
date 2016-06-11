// Steve Phillips / elimisteve
// 2015.02.24

package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/elimisteve/cryptag"
	"github.com/elimisteve/fun"
	uuid "github.com/nu7hatch/gouuid"
)

type Row struct {
	// Populated by server
	Encrypted  []byte   `json:"data"`
	RandomTags []string `json:"tags"`

	// Populated locally
	decrypted []byte
	plainTags []string
	Nonce     *[24]byte `json:"nonce"`
}

var (
	ErrRowsNotFound = errors.New("No rows found")
)

// NewRow returns a *Row containing the passed-in values in addition
// to a unique ID tag ("id:new-uuid-goes-here"), the "all" tag, and a
// new cryptographic nonce.
func NewRow(decrypted []byte, plainTags []string) (*Row, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("Error generating new UUID for Row: %v", err)
	}
	// TODO(elimisteve): Document `id:`-prefix and related conventions
	uuidTag := "id:" + id.String()

	created := "created:" + cryptag.TimeStr(cryptag.Now())

	// For future queryability-related reasons, UUID must come first!
	plainTags = append([]string{uuidTag}, append(plainTags, created, "all")...)

	nonce, err := cryptag.RandomNonce()
	if err != nil {
		return nil, err
	}

	row := &Row{decrypted: decrypted, plainTags: plainTags, Nonce: nonce}

	return row, nil
}

func NewRowSimple(decrypted []byte, plainTags []string) (*Row, error) {
	// TODO: Ensure that len(plainTags) > 0?

	nonce, err := cryptag.RandomNonce()
	if err != nil {
		return nil, err
	}

	row := &Row{decrypted: decrypted, plainTags: plainTags, Nonce: nonce}

	return row, nil
}

// NewRowFromBytes unmarshals b into a new *Row.
func NewRowFromBytes(b []byte) (*Row, error) {
	row := &Row{}
	if err := json.Unmarshal(b, row); err != nil {
		return nil, fmt.Errorf("Error creating new row: `%v`. Input: `%s`", err,
			b)
	}
	if Debug {
		log.Printf("Created new Row `%#v` from bytes: `%s`\n", row, b)
	}
	return row, nil
}

// Decrypted returns row.decrypted, row's (unexported) decrypted data (if any).
func (row *Row) Decrypted() []byte {
	return row.decrypted
}

// PlainTags returns row.plaintags, row's (unexported) plain
// (human-entered, human-readable) tags.
func (row *Row) PlainTags() []string {
	return row.plainTags
}

// HasRandomTag answers the question, "does row have the random tag randtag?"
func (row *Row) HasRandomTag(randtag string) bool {
	return fun.SliceContains(row.RandomTags, randtag)
}

// HasPlainTag answers the question, "does row have the plain tag plain?"
func (row *Row) HasPlainTag(plain string) bool {
	return fun.SliceContains(row.plainTags, plain)
}

// Format formats row for printing to the terminal.  Only suitable for
// plain text Rows.
func (row *Row) Format() string {
	return fmt.Sprintf("%s    %s\n", row.decrypted, strings.Join(row.plainTags, "   "))
}

// Decrypt sets row.decrypted, row.nonce based upon row.Encrypted,
// nonce.  The passed-in `decrypt` function will typically be
// bkend.Decrypt, where `bkend` is the backend storing this Row.
func (row *Row) Decrypt(key *[32]byte) error {
	if len(row.Encrypted) == 0 {
		if Debug {
			log.Printf("row.Decrypt: no data to decrypt, returning nil (no error)\n")
		}
		return nil
	}

	if key == nil {
		if Debug {
			log.Printf("nil key passed to row.Decrypt for row `%#v`\n", row)
		}
		return cryptag.ErrNilKey
	}

	dec, err := cryptag.Decrypt(row.Encrypted, row.Nonce, key)
	if err != nil {
		return fmt.Errorf("Error decrypting: %v", err)
	}

	row.decrypted = dec

	return nil
}

// SetPlainTags uses row.RandomTags and pairs to set row.plainTags
func (row *Row) SetPlainTags(pairs TagPairs) error {
	matches, err := pairs.WithAllRandomTags(row.RandomTags)
	if err != nil {
		return err
	}

	row.plainTags = matches.AllPlain()

	if Debug {
		log.Printf("row.plainTags set to `%#v`\n", row.plainTags)
	}

	return nil
}

// Populate sets row.decrypted based on row.Encrypted and
// row.plainTags based on row.RandomTags, thereby populating row with
// plaintext data.
func (row *Row) Populate(key *[32]byte, pairs TagPairs) error {
	if err := row.Decrypt(key); err != nil {
		return fmt.Errorf("Error decrypting row: %v", err)
	}
	if err := row.SetPlainTags(pairs); err != nil {
		return fmt.Errorf("Error setting row's plain tags: %v", err)
	}
	return nil
}
