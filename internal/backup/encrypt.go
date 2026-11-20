package backup

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	magic         = "ECOPASTEBAK2"
	maxHeaderSize = 64 * 1024

	argonMemoryKiB = 64 * 1024
	argonTimeCost  = 3
	argonThreads   = 1
	keyLength      = 32
	saltLength     = 16
	nonceLength    = 24
)

type containerHeader struct {
	Salt  []byte `json:"salt"`
	Nonce []byte `json:"nonce"`
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
	header := &containerHeader{Salt: salt, Nonce: nonce}
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
	if len(data) < len(magic) || string(data[:len(magic)]) != magic {
		return nil, errors.New(i18n.T(i18n.BackupInvalidFile))
	}
	contents := bytes.NewReader(data[len(magic):])
	header, err := readEncryptedHeader(contents)
	if err != nil {
		return nil, err
	}
	if password == "" {
		return nil, errors.New(i18n.T(i18n.BackupPasswordMissing))
	}
	key := argon2.IDKey([]byte(password), header.Salt, argonTimeCost, argonMemoryKiB, argonThreads, keyLength)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf(i18n.T(i18n.BackupInitDecrypt), err)
	}
	payload, err := aead.Open(nil, header.Nonce, data[len(data)-contents.Len():], nil)
	if err != nil {
		return nil, errors.New(i18n.T(i18n.BackupWrongPassword))
	}
	return payload, nil
}

func readEncryptedHeader(src io.Reader) (containerHeader, error) {
	var header containerHeader
	var size uint32
	if err := binary.Read(src, binary.LittleEndian, &size); err != nil || size == 0 || size > maxHeaderSize {
		return header, errors.New(i18n.T(i18n.BackupInvalidHeader))
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(src, data); err != nil {
		return header, errors.New(i18n.T(i18n.BackupInvalidHeader))
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return header, fmt.Errorf(i18n.T(i18n.BackupParseHeader), err)
	}
	if len(header.Salt) != saltLength || len(header.Nonce) != nonceLength {
		return header, errors.New(i18n.T(i18n.BackupInvalidHeader))
	}
	return header, nil
}
