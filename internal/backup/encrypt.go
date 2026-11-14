package backup

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	magic = "ECOPASTEBAK1"

	argonMemoryKiB = 64 * 1024
	argonTimeCost  = 3
	argonThreads   = 1
	keyLength      = 32
	saltLength     = 16
	nonceLength    = 24
)

type containerHeader struct {
	FormatVersion uint16        `json:"formatVersion"`
	Mode          Mode          `json:"mode"`
	KDF           *kdfHeader    `json:"kdf"`
	Cipher        *cipherHeader `json:"cipher"`
}

type kdfHeader struct {
	Algorithm   string   `json:"algorithm"`
	MemoryKiB   uint32   `json:"memoryKib"`
	TimeCost    uint32   `json:"timeCost"`
	Parallelism uint32   `json:"parallelism"`
	Salt        byteList `json:"salt"`
}

type cipherHeader struct {
	Algorithm string   `json:"algorithm"`
	Nonce     byteList `json:"nonce"`
}

// byteList keeps byte slices as JSON arrays, matching the Rust backup
// header rather than encoding/json's default base64 string.
type byteList []byte

func (b byteList) MarshalJSON() ([]byte, error) {
	values := make([]string, len(b))
	for i, value := range b {
		values[i] = strconv.Itoa(int(value))
	}
	return []byte("[" + strings.Join(values, ",") + "]"), nil
}

func (b *byteList) UnmarshalJSON(data []byte) error {
	var values []int
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	result := make([]byte, len(values))
	for i, value := range values {
		if value < 0 || value > 255 {
			return fmt.Errorf(i18n.T(i18n.BackupByteOutOfRange), value)
		}
		result[i] = byte(value)
	}
	*b = result
	return nil
}

func writeEncrypted(dst io.Writer, payload, password string) error {
	data, err := os.ReadFile(payload)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupReadPayload), err)
	}
	header, ciphertext, err := encryptPayload(data, password)
	if err != nil {
		return err
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupEncodeHeader), err)
	}
	if len(headerBytes) > 1<<20 {
		return errors.New(i18n.T(i18n.BackupHeaderTooLarge))
	}
	if _, err := io.WriteString(dst, magic); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteFile), err)
	}
	if err := binary.Write(dst, binary.LittleEndian, uint32(len(headerBytes))); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteFile), err)
	}
	if _, err := dst.Write(headerBytes); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteFile), err)
	}
	if _, err := dst.Write(ciphertext); err != nil {
		return fmt.Errorf(i18n.T(i18n.BackupWriteFile), err)
	}
	return nil
}

func encryptPayload(payload []byte, password string) (*containerHeader, []byte, error) {
	salt := make([]byte, saltLength)
	nonce := make([]byte, nonceLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf(i18n.T(i18n.BackupGenerateSalt), err)
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf(i18n.T(i18n.BackupGenerateNonce), err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTimeCost, argonMemoryKiB, argonThreads, keyLength)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, fmt.Errorf(i18n.T(i18n.BackupInitEncrypt), err)
	}
	ciphertext := aead.Seal(nil, nonce, payload, nil)
	header := &containerHeader{
		FormatVersion: formatVersion,
		Mode:          ModeEncrypted,
		KDF: &kdfHeader{
			Algorithm: "argon2id", MemoryKiB: argonMemoryKiB,
			TimeCost: argonTimeCost, Parallelism: argonThreads, Salt: salt,
		},
		Cipher: &cipherHeader{Algorithm: "xchacha20poly1305", Nonce: nonce},
	}
	return header, ciphertext, nil
}

// readPayload returns the ZIP payload of a plain or encrypted backup.
func readPayload(path, password string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.BackupReadFile), err)
	}
	return decodePayload(data, password)
}

func decodePayload(data []byte, password string) ([]byte, error) {
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		return data, nil
	}
	if len(data) < len(magic)+4 || string(data[:len(magic)]) != magic {
		return nil, errors.New(i18n.T(i18n.BackupInvalidFile))
	}
	header, headerSize, err := readHeaderWithSize(data[len(magic):])
	if err != nil {
		return nil, err
	}
	if header.Mode != ModeEncrypted || header.KDF == nil || header.Cipher == nil {
		return nil, errors.New(i18n.T(i18n.BackupInvalidHeader))
	}
	if header.KDF.Algorithm != "argon2id" || header.Cipher.Algorithm != "xchacha20poly1305" {
		return nil, errors.New(i18n.T(i18n.BackupUnsupportedEncrypt))
	}
	if len(header.KDF.Salt) != saltLength || len(header.Cipher.Nonce) != nonceLength {
		return nil, errors.New(i18n.T(i18n.BackupInvalidHeader))
	}
	if header.KDF.TimeCost < 1 || header.KDF.TimeCost > 10 ||
		header.KDF.MemoryKiB < 8*1024 || header.KDF.MemoryKiB > 256*1024 ||
		header.KDF.Parallelism < 1 || header.KDF.Parallelism > 8 {
		return nil, errors.New(i18n.T(i18n.BackupInvalidEncryption))
	}
	if password == "" {
		return nil, errors.New(i18n.T(i18n.BackupPasswordMissing))
	}
	key := argon2.IDKey([]byte(password), header.KDF.Salt,
		header.KDF.TimeCost, header.KDF.MemoryKiB, uint8(header.KDF.Parallelism), keyLength)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.BackupInitDecrypt), err)
	}
	start := len(magic) + 4 + headerSize
	payload, err := aead.Open(nil, header.Cipher.Nonce, data[start:], nil)
	if err != nil {
		return nil, errors.New(i18n.T(i18n.BackupWrongPassword))
	}
	return payload, nil
}

func readHeader(data []byte) (containerHeader, error) {
	header, _, err := readHeaderWithSize(data)
	return header, err
}

func readHeaderWithSize(data []byte) (containerHeader, int, error) {
	var header containerHeader
	if len(data) < 4 {
		return header, 0, errors.New(i18n.T(i18n.BackupInvalidHeader))
	}
	headerSize := int(binary.LittleEndian.Uint32(data[:4]))
	if headerSize <= 0 || headerSize > 64*1024 || 4+headerSize > len(data) {
		return header, 0, errors.New(i18n.T(i18n.BackupInvalidHeader))
	}
	if err := json.Unmarshal(data[4:4+headerSize], &header); err != nil {
		return header, 0, fmt.Errorf(i18n.T(i18n.BackupParseHeader), err)
	}
	if header.FormatVersion != formatVersion {
		return header, 0, fmt.Errorf(i18n.T(i18n.BackupUnsupportedVersion), header.FormatVersion)
	}
	return header, headerSize, nil
}
