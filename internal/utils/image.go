package utils

import (
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func GetImageDimensions(file multipart.File) (*ImageDimensions, error) {
	// Get current position
	currentPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	
	// Reset to beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	
	// Decode image
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, err
	}
	
	// Reset position
	_, err = file.Seek(currentPos, io.SeekStart)
	if err != nil {
		return nil, err
	}
	
	return &ImageDimensions{
		Width:  config.Width,
		Height: config.Height,
	}, nil
}

func ResizeImage(file multipart.File, filename string, maxWidth, maxHeight uint) (image.Image, error) {
	// Reset to beginning
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	
	// Decode image
	img, err := decodeImage(file, filename)
	if err != nil {
		return nil, err
	}
	
	// Get current dimensions
	bounds := img.Bounds()
	width := uint(bounds.Dx())
	height := uint(bounds.Dy())
	
	// Calculate new dimensions
	if width <= maxWidth && height <= maxHeight {
		return img, nil
	}
	
	// Calculate scaling factor
	widthRatio := float64(maxWidth) / float64(width)
	heightRatio := float64(maxHeight) / float64(height)
	
	var newWidth, newHeight uint
	if widthRatio < heightRatio {
		newWidth = maxWidth
		newHeight = uint(float64(height) * widthRatio)
	} else {
		newWidth = uint(float64(width) * heightRatio)
		newHeight = maxHeight
	}
	
	// Resize image
	resized := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
	
	return resized, nil
}

func decodeImage(file multipart.File, filename string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(file)
	case ".png":
		return png.Decode(file)
	default:
		// Try generic decode
		img, _, err := image.Decode(file)
		return img, err
	}
}

func EncodeImage(img image.Image, format string, writer io.Writer, quality int) error {
	switch strings.ToLower(format) {
	case "jpg", "jpeg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: quality})
	case "png":
		return png.Encode(writer, img)
	default:
		return errors.New("unsupported image format")
	}
}

func ValidateImageDimensions(file multipart.File, minWidth, minHeight, maxWidth, maxHeight int) error {
	dimensions, err := GetImageDimensions(file)
	if err != nil {
		return err
	}
	
	if dimensions.Width < minWidth || dimensions.Height < minHeight {
		return errors.New("image dimensions too small")
	}
	
	if dimensions.Width > maxWidth || dimensions.Height > maxHeight {
		return errors.New("image dimensions too large")
	}
	
	return nil
}

func IsValidImageFormat(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validFormats := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	
	for _, format := range validFormats {
		if ext == format {
			return true
		}
	}
	
	return false
}

func GenerateThumbnail(file multipart.File, filename string) (image.Image, error) {
	return ResizeImage(file, filename, 150, 150)
}