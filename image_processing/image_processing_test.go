package imageprocessing

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// makeTestImage builds a small solid-color image in memory,
// so the tests don't depend on any photo files.
func makeTestImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestWriteThenReadImage(t *testing.T) {
	dir := t.TempDir() // a temporary folder Go deletes after the test
	path := filepath.Join(dir, "test.jpeg")

	original := makeTestImage(40, 20, color.RGBA{255, 0, 0, 255})
	if err := WriteImage(path, original); err != nil {
		t.Fatalf("WriteImage returned an error: %v", err)
	}

	img, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage returned an error: %v", err)
	}

	got := img.Bounds()
	if got.Dx() != 40 || got.Dy() != 20 {
		t.Errorf("expected a 40x20 image, got %dx%d", got.Dx(), got.Dy())
	}
}

func TestReadImageMissingFile(t *testing.T) {
	_, err := ReadImage("this_file_does_not_exist.jpeg")
	if err == nil {
		t.Error("expected an error for a missing file, got nil")
	}
}

func TestReadImageNotAnImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.jpeg")
	if err := os.WriteFile(path, []byte("this is not an image"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadImage(path)
	if err == nil {
		t.Error("expected an error for a file that isn't an image, got nil")
	}
}

func TestWriteImageCreatesFolder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new_folder", "test.jpeg")

	if err := WriteImage(path, makeTestImage(10, 10, color.White)); err != nil {
		t.Fatalf("WriteImage returned an error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist at %s: %v", path, err)
	}
}

func TestWriteImageNilImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nil.jpeg")
	if err := WriteImage(path, nil); err == nil {
		t.Error("expected an error when writing a nil image, got nil")
	}
}

func TestResize(t *testing.T) {
	tests := []struct {
		name                  string
		width, height         int
		wantWidth, wantHeight int
	}{
		{"landscape", 1000, 800, 500, 400},
		{"portrait", 600, 1200, 250, 500},
		{"square", 800, 800, 500, 500},
	}

	for _, tc := range tests {
		img := makeTestImage(tc.width, tc.height, color.White)
		got := Resize(img).Bounds()

		if got.Dx() != tc.wantWidth || got.Dy() != tc.wantHeight {
			t.Errorf("%s: expected %dx%d, got %dx%d",
				tc.name, tc.wantWidth, tc.wantHeight, got.Dx(), got.Dy())
		}
	}
}

func TestGrayscale(t *testing.T) {
	img := makeTestImage(10, 10, color.RGBA{255, 0, 0, 255}) // pure red
	gray := Grayscale(img)

	// The result should be a grayscale image type
	grayImg, ok := gray.(*image.Gray)
	if !ok {
		t.Fatalf("expected *image.Gray, got %T", gray)
	}

	// Size should not change
	if grayImg.Bounds() != img.Bounds() {
		t.Errorf("expected bounds %v, got %v", img.Bounds(), grayImg.Bounds())
	}

	// Pure red should turn into a medium-dark gray, not black or white
	pixel := grayImg.GrayAt(5, 5).Y
	if pixel == 0 || pixel == 255 {
		t.Errorf("expected a gray value between 0 and 255, got %d", pixel)
	}
}
