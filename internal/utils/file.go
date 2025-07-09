package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

func IsAllowedFileType(filename string, allowedTypes []string) bool {
	ext := strings.TrimPrefix(GetFileExtension(filename), ".")
	
	for _, allowedType := range allowedTypes {
		if ext == allowedType {
			return true
		}
	}
	
	return false
}

func IsImageFile(filename string) bool {
	return IsAllowedFileType(filename, AllowedImageTypes)
}

func IsDocumentFile(filename string) bool {
	return IsAllowedFileType(filename, AllowedDocumentTypes)
}

func IsAudioFile(filename string) bool {
	return IsAllowedFileType(filename, AllowedAudioTypes)
}

func IsVideoFile(filename string) bool {
	return IsAllowedFileType(filename, AllowedVideoTypes)
}

func GenerateUniqueFilename(originalFilename string) string {
	ext := GetFileExtension(originalFilename)
	timestamp := time.Now().Unix()
	randomStr := GenerateRandomString(8)
	
	return fmt.Sprintf("%d_%s%s", timestamp, randomStr, ext)
}

func GetFileSizeFromHeader(file multipart.File) (int64, error) {
	// Get current position
	currentPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	
	// Get file size
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	
	// Reset position
	_, err = file.Seek(currentPos, io.SeekStart)
	if err != nil {
		return 0, err
	}
	
	return size, nil
}

func ValidateFileSize(file multipart.File, maxSize int64) error {
	size, err := GetFileSizeFromHeader(file)
	if err != nil {
		return err
	}
	
	if size > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", size, maxSize)
	}
	
	return nil
}

func CreateDirectory(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func DeleteFile(path string) error {
	if FileExists(path) {
		return os.Remove(path)
	}
	return nil
}

func GetFileMD5(file multipart.File) (string, error) {
	hash := md5.New()
	
	// Get current position
	currentPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}
	
	// Reset to beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}
	
	// Copy file to hash
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	
	// Reset position
	_, err = file.Seek(currentPos, io.SeekStart)
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func GetContentType(filename string) string {
	ext := GetFileExtension(filename)
	
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":  "text/plain",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".aac":  "audio/aac",
		".m4a":  "audio/mp4",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
	}
	
	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}
	
	return "application/octet-stream"
}
