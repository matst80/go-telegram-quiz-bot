package tts

import (
	"errors"
	"fmt"
	"os"

	"github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// PiperConfig holds the paths needed to initialize a Piper VITS model.
type PiperConfig struct {
	ModelPath  string // Path to the .onnx model file
	TokensPath string // Path to tokens.txt
	DataDir    string // Path to espeak-ng-data directory
}

type piperService struct {
	tts *sherpa_onnx.OfflineTts
}

// NewPiperService creates a new TTS Service using the sherpa-onnx Piper VITS implementation.
func NewPiperService(config PiperConfig) (Service, error) {
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", config.ModelPath)
	}
	if _, err := os.Stat(config.TokensPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tokens file not found: %s", config.TokensPath)
	}
	if _, err := os.Stat(config.DataDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("data dir not found: %s", config.DataDir)
	}

	ttsConfig := &sherpa_onnx.OfflineTtsConfig{
		Model: sherpa_onnx.OfflineTtsModelConfig{
			Vits: sherpa_onnx.OfflineTtsVitsModelConfig{
				Model:       config.ModelPath,
				Tokens:      config.TokensPath,
				DataDir:     config.DataDir,
				NoiseScale:  0.667,
				NoiseScaleW: 0.8,
				LengthScale: 1.0,
			},
			NumThreads: 1,
			Debug:      0,
			Provider:   "cpu",
		},
	}

	tts := sherpa_onnx.NewOfflineTts(ttsConfig)
	if tts == nil {
		return nil, errors.New("failed to initialize sherpa-onnx OfflineTts")
	}

	return &piperService{
		tts: tts,
	}, nil
}

func (s *piperService) GenerateSpeech(text string, outputPath string) error {
	if s.tts == nil {
		return errors.New("tts service is not initialized")
	}

	// Generate audio with speaker ID 0 and speed 1.0
	audio := s.tts.Generate(text, 0, 1.0)
	if audio == nil {
		return errors.New("failed to generate audio")
	}

	ok := audio.Save(outputPath)
	if !ok {
		return fmt.Errorf("failed to save audio to %s", outputPath)
	}

	return nil
}

// Close should be called if we implement a closer, but sherpa_onnx provides DeleteOfflineTts.
// We can add a Close method if needed in the future.
func (s *piperService) Close() {
	if s.tts != nil {
		sherpa_onnx.DeleteOfflineTts(s.tts)
		s.tts = nil
	}
}
