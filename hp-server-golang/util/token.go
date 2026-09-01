package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"hp-server-lib/log"
	"io"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

var aes_key []byte

// init 每次启动随机生成 AES-256 密钥，重启后之前签发的 token 自动失效
func init() {
	aes_key = make([]byte, 32)
	_, err := rand.Read(aes_key)
	if err != nil {
		panic(fmt.Sprintf("生成随机密钥失败: %v", err))
	}
}

// AES 加密
func aesEncrypt(plainText, key []byte) (string, error) {
	// 生成 AES 块加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 生成随机的 IV（初始化向量）
	ciphertext := make([]byte, aes.BlockSize+len(plainText))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	// 创建加密模式
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plainText)

	// 返回 Base64 编码的加密数据
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// AES 解密
func aesDecrypt(cipherTextBase64, key []byte) (string, error) {
	// 解码 Base64
	cipherText, err := base64.URLEncoding.DecodeString(string(cipherTextBase64))
	if err != nil {
		return "", err
	}

	// 生成 AES 块加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 提取 IV 和加密数据
	if len(cipherText) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	// 创建解密模式
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	// 返回解密后的明文
	return string(cipherText), nil
}

// 生成 Token（加密）
func GenerateToken(userId, role string) (string, error) {
	// 获取当前时间戳（毫秒）
	timestamp := time.Now().UnixMilli()

	// 拼接 userId + role + timestamp
	plainText := fmt.Sprintf("%s|%s|%d", userId, role, timestamp)

	// AES 加密并生成 Base64 编码的 Token
	return aesEncrypt([]byte(plainText), aes_key)
}

// 解密 Token
func DecodeToken(token string) (resultNum int, resultRole string, resultTs int64, err error) {
	// 关键：旧实现 recover 后不修改返回值，导致 panic 也会以 err=nil 返回，
	// 调用方拿到 0/""/0 当合法 token 处理，等于鉴权绕过。这里用命名返回值 + recover 透传。
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("解析Token错误: %v\n栈情况: %s", r, string(debug.Stack()))
			resultNum = 0
			resultRole = ""
			resultTs = 0
			err = fmt.Errorf("token 解析异常: %v", r)
		}
	}()

	// 解密 Base64 编码的 Token
	decodedText, decErr := aesDecrypt([]byte(token), aes_key)
	if decErr != nil {
		return 0, "", 0, decErr
	}
	parts := strings.Split(decodedText, "|")
	// 显式校验长度，避免对空切片 / 越界 panic
	if len(parts) < 3 {
		return 0, "", 0, fmt.Errorf("token 格式错误")
	}
	num, atoiErr := strconv.Atoi(parts[0])
	if atoiErr != nil {
		return 0, "", 0, fmt.Errorf("token userId 解析失败: %w", atoiErr)
	}
	num2, piErr := strconv.ParseInt(parts[2], 10, 64)
	if piErr != nil {
		return 0, parts[1], 0, fmt.Errorf("token timestamp 解析失败: %w", piErr)
	}
	return num, parts[1], num2, nil
}
