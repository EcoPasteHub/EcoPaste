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
			return fmt.Errorf("备份字节值 %d 越界", value)
		}
		result[i] = byte(value)
	}
	*b = result
	return nil
}

func writeEncrypted(dst io.Writer, payload, password string) error {
	data, err := os.ReadFile(payload)
	if err != nil {
		return fmt.Errorf("读取备份载荷: %w", err)
	}
	header, ciphertext, err := encryptPayload(data, password)
	if err != nil {
		return err
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("编码备份文件头: %w", err)
	}
	if len(headerBytes) > 1<<20 {
		return errors.New("备份文件头过大")
	}
	if _, err := io.WriteString(dst, magic); err != nil {
		return fmt.Errorf("写入备份文件: %w", err)
	}
	if err := binary.Write(dst, binary.LittleEndian, uint32(len(headerBytes))); err != nil {
		return fmt.Errorf("写入备份文件: %w", err)
	}
	if _, err := dst.Write(headerBytes); err != nil {
		return fmt.Errorf("写入备份文件: %w", err)
	}
	if _, err := dst.Write(ciphertext); err != nil {
		return fmt.Errorf("写入备份文件: %w", err)
	}
	return nil
}

func encryptPayload(payload []byte, password string) (*containerHeader, []byte, error) {
	salt := make([]byte, saltLength)
	nonce := make([]byte, nonceLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf("生成备份盐: %w", err)
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("生成备份随机数: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTimeCost, argonMemoryKiB, argonThreads, keyLength)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, fmt.Errorf("初始化备份加密: %w", err)
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
		return nil, fmt.Errorf("读取备份文件: %w", err)
	}
	return decodePayload(data, password)
}

func decodePayload(data []byte, password string) ([]byte, error) {
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		return data, nil
	}
	if len(data) < len(magic)+4 || string(data[:len(magic)]) != magic {
		return nil, errors.New("不是有效的 EcoPaste 备份文件")
	}
	headerSize := int(binary.LittleEndian.Uint32(data[len(magic) : len(magic)+4]))
	start := len(magic) + 4
	if headerSize <= 0 || start+headerSize > len(data) {
		return nil, errors.New("加密备份文件头无效")
	}
	var header containerHeader
	if err := json.Unmarshal(data[start:start+headerSize], &header); err != nil {
		return nil, fmt.Errorf("解析备份文件头: %w", err)
	}
	if header.FormatVersion != formatVersion {
		return nil, fmt.Errorf("暂不支持备份格式版本 %d", header.FormatVersion)
	}
	if header.Mode != ModeEncrypted || header.KDF == nil || header.Cipher == nil {
		return nil, errors.New("加密备份文件头无效")
	}
	if header.KDF.Algorithm != "argon2id" || header.Cipher.Algorithm != "xchacha20poly1305" {
		return nil, errors.New("暂不支持该备份加密格式")
	}
	if len(header.KDF.Salt) != saltLength || len(header.Cipher.Nonce) != nonceLength {
		return nil, errors.New("加密备份文件头无效")
	}
	if header.KDF.TimeCost < 1 || header.KDF.TimeCost > 10 ||
		header.KDF.MemoryKiB < 8*1024 || header.KDF.MemoryKiB > 256*1024 ||
		header.KDF.Parallelism < 1 || header.KDF.Parallelism > 8 {
		return nil, errors.New("加密备份参数无效")
	}
	if password == "" {
		return nil, errors.New("请输入备份密码")
	}
	key := argon2.IDKey([]byte(password), header.KDF.Salt,
		header.KDF.TimeCost, header.KDF.MemoryKiB, uint8(header.KDF.Parallelism), keyLength)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("初始化备份解密: %w", err)
	}
	payload, err := aead.Open(nil, header.Cipher.Nonce, data[start+headerSize:], nil)
	if err != nil {
		return nil, errors.New("备份密码不正确或文件已损坏")
	}
	return payload, nil
}
