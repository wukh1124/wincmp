package crypto

import (
	"encoding/base64"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	modcrypt32  = syscall.NewLazyDLL("crypt32.dll")
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCryptProtectData   = modcrypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = modcrypt32.NewProc("CryptUnprotectData")
	procLocalFree          = modkernel32.NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newDataBlob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
}

func blobToSlice(blob dataBlob) []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return nil
	}
	out := make([]byte, blob.cbData)
	copy(out, (*[1 << 28]byte)(unsafe.Pointer(blob.pbData))[:blob.cbData:blob.cbData])
	return out
}

// Encrypt 使用 Windows DPAPI 加密資料，回傳 base64 編碼字串
// 前綴 "ENC:" 用於標識已加密的密碼
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	data := []byte(plaintext)
	inBlob := newDataBlob(data)
	var outBlob dataBlob

	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if ret == 0 {
		return "", fmt.Errorf("CryptProtectData 失敗: %w", err)
	}
	defer localFree(uintptr(unsafe.Pointer(outBlob.pbData)))

	encrypted := blobToSlice(outBlob)
	return "ENC:" + base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt 使用 Windows DPAPI 解密資料，接受 base64 編碼字串
// 若字串不含 "ENC:" 前綴，視為明文直接回傳（向後相容）
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	if !isEncrypted(ciphertext) {
		return ciphertext, nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext[4:])
	if err != nil {
		return "", fmt.Errorf("base64 解碼失敗: %w", err)
	}

	inBlob := newDataBlob(data)
	var outBlob dataBlob

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if ret == 0 {
		return "", fmt.Errorf("CryptUnprotectData 失敗: %w", err)
	}
	defer localFree(uintptr(unsafe.Pointer(outBlob.pbData)))

	decrypted := blobToSlice(outBlob)
	return string(decrypted), nil
}

// IsEncrypted 檢查字串是否為 DPAPI 加密格式
func IsEncrypted(s string) bool {
	return isEncrypted(s)
}

func isEncrypted(s string) bool {
	return len(s) > 4 && s[:4] == "ENC:"
}

func localFree(ptr uintptr) {
	procLocalFree.Call(ptr)
	runtime.KeepAlive(ptr)
}
