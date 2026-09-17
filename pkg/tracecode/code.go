package tracecode

import (
	"crypto/rand"
	"fmt"
	"time"
)

const (
	codeAlphabet = "0123456789ABCDEFGHJKLMNPQRSTUVWXYZ" // Base32 (no I,O)
	codeLength   = 10
)

// Generate creates a trace code using millisecond timestamp + seq
// Format: 9 data chars + 1 checksum char = 10 chars
func Generate(seq int64) string {
	nowMs := time.Now().UnixMilli()

	// Build 10 bytes with seq mixed into all positions
	data := make([]byte, 10)
	// Distribute timestamp bytes
	data[0] = byte(nowMs >> 40)
	data[1] = byte(nowMs >> 32)
	data[2] = byte(nowMs >> 24)
	data[3] = byte(nowMs >> 16)
	data[4] = byte(nowMs >> 8)
	data[5] = byte(nowMs)
	// Distribute seq bytes
	data[6] = byte(seq >> 56)
	data[7] = byte(seq >> 48)
	data[8] = byte(seq >> 40)
	data[9] = byte(seq >> 32)

	// Mix seq bytes into timestamp bytes
	data[0] ^= byte(seq)
	data[1] ^= byte(seq >> 8)
	data[2] ^= byte(seq >> 16)
	data[3] ^= byte(seq >> 24)
	data[4] ^= byte(seq >> 32)
	data[5] ^= byte(seq >> 40)

	// Mix timestamp bytes into seq bytes
	data[6] ^= byte(nowMs)
	data[7] ^= byte(nowMs >> 8)
	data[8] ^= byte(nowMs >> 16)
	data[9] ^= byte(nowMs >> 24)

	// Encode to base32 (10 bytes -> 16 chars)
	encoded := base32Encode(data)
	code := encoded[:9]

	// CRC8 checksum
	checksum := crc8([]byte(code))
	code = code + string(codeAlphabet[checksum%32])

	return code
}

// Validate checks if a trace code has valid checksum
func Validate(code string) bool {
	if len(code) != codeLength {
		return false
	}
	data := code[:9]
	expectedCRC := crc8([]byte(data))
	return code[9] == codeAlphabet[expectedCRC%32]
}

func base32Encode(data []byte) string {
	var result []byte
	for i := 0; i < len(data); i += 5 {
		buf := make([]byte, 5)
		copy(buf, data[i:])
		var val uint64
		for j := 0; j < 5; j++ {
			val = (val << 8) | uint64(buf[j])
		}
		for j := 8; j > 0; j-- {
			result = append(result, codeAlphabet[(val>>uint(5*(j-1)))&31])
		}
	}
	return string(result)
}

func crc8(data []byte) byte {
	var crc byte
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x07
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// RandomUUID returns a random UUID string for client_uuid
func RandomUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}