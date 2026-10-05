package sceneanalysis

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"unicode/utf8"
)

const MaxSCEBytes = 16 << 20
const SemanticVersion = "sce-declared-v2"

type Container struct {
	Format        string `json:"format"`
	Version       uint32 `json:"version,omitempty"`
	BodySHA256    string `json:"bodySHA256"`
	TrailerBytes  int    `json:"trailerBytes,omitempty"`
	TrailerSHA256 string `json:"trailerSHA256,omitempty"`
}

func digest(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

func containerSignature(c *Container) string {
	if c == nil {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d:%d:%s", c.Format, c.Version, c.TrailerBytes, c.TrailerSHA256)
}

// Decode accepts only UTF-8 configuration or the observed Workshop v1 container.
// The bounded UTF-8 body is decoded; an opaque trailer is preserved as metadata.
// It does not search arbitrary .bin files for apparent configuration strings.
func Decode(data []byte) ([]byte, Container, error) {
	if len(data) == 0 || len(data) > MaxSCEBytes {
		return nil, Container{}, fmt.Errorf("SCE 为空或超过 16 MB")
	}
	c := Container{Format: "utf8_text"}
	body := data
	if len(data) >= 4 && binary.LittleEndian.Uint32(data[:4]) == 0xf55bace6 {
		if len(data) < 12 {
			return nil, c, fmt.Errorf("SCE 二进制文件头不完整")
		}
		version := binary.LittleEndian.Uint32(data[4:8])
		length := uint64(binary.LittleEndian.Uint32(data[8:12]))
		if version != 1 {
			return nil, c, fmt.Errorf("未知 SCE 封装版本 %d", version)
		}
		if length < 2 || length > uint64(len(data)-12) {
			return nil, c, fmt.Errorf("SCE 声明正文长度无效")
		}
		end := 12 + int(length)
		if data[end-1] != 0 {
			return nil, c, fmt.Errorf("SCE 正文终止标记缺失")
		}
		body = data[12 : end-1]
		c.Format = "workshop_binary_v1"
		c.Version = version
		c.TrailerBytes = len(data) - end
		if c.TrailerBytes > 0 {
			c.TrailerSHA256 = digest(data[end:])
		}
	}
	if bytes.IndexByte(body, 0) >= 0 || !utf8.Valid(body) {
		return nil, c, fmt.Errorf("SCE 正文不是有效 UTF-8 配置")
	}
	c.BodySHA256 = digest(body)
	return body, c, nil
}
