package tts

// Service defines the interface for Text-to-Speech generation.
type Service interface {
	// GenerateSpeech generates audio from the given text and saves it to outputPath.
	// Returns an error if the generation or saving fails.
	GenerateSpeech(text string, outputPath string) error
}
