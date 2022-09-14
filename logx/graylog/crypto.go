package graylog

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/zenazn/pkcs7pad"
)

// cryptoInterface
type cryptoInterface interface {
	encrypt(data []byte) ([]byte, error)
}

// md5 crypto
type md5Crypto struct{}

// new md5 crypto
func newMD5Crypto() *md5Crypto {
	return &md5Crypto{}
}

//encrypt
func (e *md5Crypto) encrypt(data []byte) ([]byte, error) {
	h := md5.New()
	h.Write(data)
	src := h.Sum(nil)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)

	return dst, nil
}

// aes crypto
type aesCrypto struct {
	key []byte
	iv  []byte
}

// new aes crypto
func newAESCrypto(key, iv []byte) *aesCrypto {
	return &aesCrypto{key: key, iv: iv}
}

func (ae *aesCrypto) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(ae.key)
	if err != nil {
		return nil, err
	}

	data = pkcs7pad.Pad(data, aes.BlockSize)
	dst := make([]byte, len(data))
	blockMode := cipher.NewCBCEncrypter(block, ae.iv)
	blockMode.CryptBlocks(dst, data)
	tdst := make([]byte, base64.StdEncoding.EncodedLen(len(dst)))
	base64.StdEncoding.Encode(tdst, dst)
	return tdst, nil
}

// EncryptField encrypt a field separated
func EncryptField(field string) string {
	if field == "" {
		return ""
	}

	var cryptor cryptoInterface
	if logOpts != nil && logOpts.cryptor != nil {
		cryptor = logOpts.cryptor
	}

	if cryptor == nil {
		return field
	}

	encryptInfo, err := cryptor.encrypt([]byte(field))
	if err != nil {
		return field
	}
	return "§" + string(encryptInfo) + "§"
}

func EncryptContent(content string) string {
	if logOpts == nil {
		return ""
	}

	encryptFields := logOpts.encryptFields
	encryptDepth := logOpts.encryptDepth
	cryptor := logOpts.cryptor

	trans := "\\"
	for _, rule := range encryptFields {
		for i := 0; i < encryptDepth+1; i++ {
			// TODO 优化字符串处理性能
			actTrans := "\""
			idxActTrans := actTrans
			for j := 0; j < i*2; j++ {
				actTrans = trans + actTrans
			}
			for k := 0; k < i; k++ {
				idxActTrans = trans + idxActTrans
			}
			reg := regexp.MustCompile(actTrans +
				rule + actTrans +
				" ?: ?" + actTrans +
				"[^" + actTrans +
				"]*" + actTrans)

			content = reg.ReplaceAllStringFunc(content,
				func(field string) string {
					var info, infoPre, infoVal, infoAft string
					iEnd := strings.LastIndex(field, idxActTrans)
					if iEnd != -1 {
						info = field[:iEnd]
						infoAft = field[iEnd:]
						iStart := strings.LastIndex(info, idxActTrans)
						if iStart != -1 {
							infoVal = info[iStart+len(idxActTrans):]
							infoPre = info[:iStart+len(idxActTrans)]
						}
					}

					// skip when empty
					if infoVal == "" {
						return field
					}

					encryptInfo, err := cryptor.encrypt([]byte(infoVal))
					if err != nil {
						return field
					}
					return infoPre + "§" + string(encryptInfo) + "§" + infoAft
				})
		}
	}
	return content
}
