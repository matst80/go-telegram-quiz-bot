package tts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPiperTTSGeneration(t *testing.T) {
	// Setup test data directory
	testdataDir := "testdata"
	err := os.MkdirAll(testdataDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create testdata dir: %v", err)
	}

	// Ensure the model is downloaded and extracted
	piperConfig, err := EnsureDefaultModel(testdataDir)
	if err != nil {
		t.Fatalf("Failed to ensure default model: %v", err)
	}

	service, err := NewPiperService(piperConfig)
	if err != nil {
		t.Fatalf("Failed to create Piper service: %v", err)
	}
	defer func() {
		if p, ok := service.(*piperService); ok {
			p.Close()
		}
	}()

	outputFile := filepath.Join(testdataDir, "output.wav")

	// Test phrase
	text := "Hola, ¿cómo estás? Esto es una prueba de síntesis de voz."

	t.Logf("Generating speech for: %q", text)
	err = service.GenerateSpeech(text, outputFile)
	if err != nil {
		t.Fatalf("Failed to generate speech: %v", err)
	}

	// Verify output
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if info.Size() == 0 {
		t.Errorf("Generated audio file is empty")
	}

	t.Logf("Successfully generated speech to %s (size: %d bytes)", outputFile, info.Size())
}
