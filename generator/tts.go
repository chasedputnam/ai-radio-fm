package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/user/go-kokoro-tts/pkg/api"
	"github.com/user/go-kokoro-tts/pkg/audio"
	"github.com/user/go-kokoro-tts/pkg/model"
	"github.com/user/go-kokoro-tts/pkg/text"
	"github.com/user/go-kokoro-tts/pkg/voice"
)

// maxChunkSize matches the safe limit used by the go-kokoro-tts CLI (~510 tokens).
const maxChunkSize = 500

type KokoroTTS struct {
	pipeline *api.Pipeline
	engine   *model.ONNXEngine
	vLoader  *voice.BinaryVoiceLoader
	chunker  *text.Chunker

	modelPath string
	voiceDir  string
}

func NewKokoroTTS(libPath, modelPath, vocabPath, voiceDir string, opts model.EngineOptions) (*KokoroTTS, error) {
	engine, err := model.NewONNXEngine(libPath, modelPath, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load onnx engine: %w", err)
	}

	var vocab map[rune]int64
	if vocabPath != "" {
		var err error
		vocab, err = text.LoadVocabulary(vocabPath)
		if err != nil {
			engine.Close()
			return nil, fmt.Errorf("failed to load vocabulary: %w", err)
		}
	}

	pipeline := api.NewPipeline(
		text.NewBasicEnglishNormalizer(),
		text.NewESpeakPhonemizer(),
		text.NewTokenizer(vocab),
		engine,
	)

	vLoader := voice.NewBinaryVoiceLoader(voiceDir)

	return &KokoroTTS{
		pipeline:  pipeline,
		engine:    engine,
		vLoader:   vLoader,
		chunker:   text.NewChunker(maxChunkSize),
		modelPath: modelPath,
		voiceDir:  voiceDir,
	}, nil
}

func (k *KokoroTTS) Render(ctx context.Context, textStr, voiceName, outputPath string) error {
	// Check context cancellation early
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	profile, err := k.vLoader.Load(voiceName)
	if err != nil {
		return fmt.Errorf("failed to load voice profile '%s': %w", voiceName, err)
	}

	// Split text into chunks to stay within the model's token limit (~510 tokens).
	// Each chunk is synthesized independently and the resulting audio is concatenated.
	chunks := k.chunker.Chunk(textStr)

	var allSamples []float32
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		// Respect context cancellation between chunks.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		samples, err := k.pipeline.Synthesize(chunk, "en-us", profile, 1.0)
		if err != nil {
			return fmt.Errorf("synthesis failed on chunk %d/%d: %w", i+1, len(chunks), err)
		}
		allSamples = append(allSamples, samples...)
	}

	// Check context cancellation before writing.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Kokoro generates at 24kHz.
	if err := audio.WriteWAV(outputPath, allSamples, 24000); err != nil {
		return fmt.Errorf("failed to write wav file: %w", err)
	}

	return nil
}

func (k *KokoroTTS) Close() error {
	if k.engine != nil {
		k.engine.Close()
	}
	return nil
}
