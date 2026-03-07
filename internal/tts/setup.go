package tts

import (
	"archive/tar"
	"compress/bzip2"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const ModelURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-piper-es_AR-daniela-high.tar.bz2"
const ModelName = "vits-piper-es_AR-daniela-high"

// DownloadFile downloads a URL to a local file.
func DownloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

// ExtractTarBz2 extracts a .tar.bz2 file to a destination folder.
func ExtractTarBz2(archive string, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	bz2Reader := bzip2.NewReader(f)
	tarReader := tar.NewReader(bz2Reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// EnsureDefaultModel downloads and extracts the default Spanish Piper model to dataDir.
// It returns a PiperConfig initialized with the paths inside dataDir.
func EnsureDefaultModel(dataDir string) (PiperConfig, error) {
	modelFolder := filepath.Join(dataDir, ModelName)
	if _, err := os.Stat(modelFolder); os.IsNotExist(err) {
		log.Printf("Downloading model from %s...", ModelURL)
		archivePath := filepath.Join(dataDir, ModelName+".tar.bz2")

		err := DownloadFile(archivePath, ModelURL)
		if err != nil {
			return PiperConfig{}, fmt.Errorf("failed to download model: %w", err)
		}
		defer os.Remove(archivePath)

		log.Println("Extracting model...")
		err = ExtractTarBz2(archivePath, dataDir)
		if err != nil {
			return PiperConfig{}, fmt.Errorf("failed to extract model: %w", err)
		}
		log.Println("Model extracted successfully.")
	}

	return PiperConfig{
		ModelPath:  filepath.Join(modelFolder, "es_AR-daniela-high.onnx"),
		TokensPath: filepath.Join(modelFolder, "tokens.txt"),
		DataDir:    filepath.Join(modelFolder, "espeak-ng-data"),
	}, nil
}
