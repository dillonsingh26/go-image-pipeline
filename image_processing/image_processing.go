package imageprocessing

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

// ReadImage opens and decodes an image file.
// It returns an error instead of crashing if the file is missing or not a valid image.
func ReadImage(path string) (image.Image, error) {
	inputFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer inputFile.Close()

	img, _, err := image.Decode(inputFile)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	return img, nil
}

// WriteImage encodes an image as a JPEG and saves it to path.
// It creates the output folder if it doesn't exist yet.
func WriteImage(path string, img image.Image) error {
	if img == nil {
		return fmt.Errorf("writing %s: image is nil", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating folder for %s: %w", path, err)
	}

	outputFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}

	if err := jpeg.Encode(outputFile, img, nil); err != nil {
		outputFile.Close()
		return fmt.Errorf("encoding %s: %w", path, err)
	}

	if err := outputFile.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	return nil
}

// Grayscale converts every pixel of the image to a shade of gray.
func Grayscale(img image.Image) image.Image {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalPixel := img.At(x, y)
			grayPixel := color.GrayModel.Convert(originalPixel)
			grayImg.Set(x, y, grayPixel)
		}
	}
	return grayImg
}

// Resize scales the image so its longest side is 500 pixels,
// keeping the original aspect ratio so nothing gets stretched or squished.
func Resize(img image.Image) image.Image {
	const maxSize = 500.0

	// Detect the input size in pixels
	bounds := img.Bounds()
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())

	// Scale both sides by the same factor
	var newWidth, newHeight float64
	if width >= height { // landscape or square: width becomes 500
		newWidth = maxSize
		newHeight = height * maxSize / width
	} else { // portrait: height becomes 500
		newHeight = maxSize
		newWidth = width * maxSize / height
	}

	return resize.Resize(uint(newWidth), uint(newHeight), img, resize.Lanczos3)
}
