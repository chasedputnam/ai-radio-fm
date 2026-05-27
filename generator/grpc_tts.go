package generator

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	ttspb "github.com/chasedputnam/go-kokoro-tts/proto/tts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/chasedputnam/go-kokoro-tts/pkg/audio"
)

// GRPCTTSRenderer implements TTSRenderer by calling the tts-server gRPC sidecar.
// Audio samples are returned as raw little-endian float32 bytes and written
// locally as a WAV file, keeping the sidecar stateless.
type GRPCTTSRenderer struct {
	client ttspb.TTSServiceClient
	conn   *grpc.ClientConn
}

// NewGRPCTTSRenderer dials the tts-server at addr and returns a ready renderer.
// Returns an error if the connection cannot be established within 5 seconds.
func NewGRPCTTSRenderer(addr string) (*GRPCTTSRenderer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, //nolint:staticcheck // DialContext deprecated in grpc v2 but fine for v1
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to TTS sidecar at %q: %w", addr, err)
	}

	return &GRPCTTSRenderer{
		client: ttspb.NewTTSServiceClient(conn),
		conn:   conn,
	}, nil
}

// Render synthesizes text using the remote TTS sidecar and writes the result
// as a WAV file at outputPath. A 120-second deadline is applied to the RPC
// to accommodate queuing behind other stations on a shared sidecar.
func (g *GRPCTTSRenderer) Render(ctx context.Context, textStr, voiceName, outputPath string) error {
	// Apply a 120-second deadline — long enough for a full script under load.
	rpcCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	resp, err := g.client.Synthesize(rpcCtx, &ttspb.SynthesizeRequest{
		Text:      textStr,
		VoiceName: voiceName,
		LangCode:  "en-us",
		Speed:     1.0,
	})
	if err != nil {
		return fmt.Errorf("TTS gRPC Synthesize failed: %w", err)
	}

	samples, err := decodePCM(resp.AudioPcm)
	if err != nil {
		return fmt.Errorf("failed to decode PCM response: %w", err)
	}

	sampleRate := int(resp.SampleRate)
	if sampleRate == 0 {
		sampleRate = 24000
	}

	if err := audio.WriteWAV(outputPath, samples, sampleRate); err != nil {
		return fmt.Errorf("failed to write WAV file: %w", err)
	}

	return nil
}

// Close releases the underlying gRPC connection.
func (g *GRPCTTSRenderer) Close() error {
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

// decodePCM converts raw little-endian float32 bytes back to []float32.
func decodePCM(pcm []byte) ([]float32, error) {
	if len(pcm)%4 != 0 {
		return nil, fmt.Errorf("PCM byte length %d is not a multiple of 4", len(pcm))
	}
	samples := make([]float32, len(pcm)/4)
	for i := range samples {
		bits := binary.LittleEndian.Uint32(pcm[i*4:])
		samples[i] = math.Float32frombits(bits)
	}
	return samples, nil
}
