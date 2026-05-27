package generator

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"os"
	"path/filepath"
	"testing"

	ttspb "github.com/chasedputnam/go-kokoro-tts/proto/tts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// mockTTSServer is an in-process gRPC server returning known PCM bytes.
type mockTTSServer struct {
	ttspb.UnimplementedTTSServiceServer
	samples []float32
}

func (m *mockTTSServer) Synthesize(_ context.Context, req *ttspb.SynthesizeRequest) (*ttspb.SynthesizeResponse, error) {
	pcm := make([]byte, len(m.samples)*4)
	for i, s := range m.samples {
		binary.LittleEndian.PutUint32(pcm[i*4:], math.Float32bits(s))
	}
	return &ttspb.SynthesizeResponse{
		AudioPcm:   pcm,
		SampleRate: 24000,
	}, nil
}

// startMockTTSServer starts an in-process gRPC server over a bufconn listener
// and returns a connected GRPCTTSRenderer.
func startMockTTSServer(t *testing.T, samples []float32) *GRPCTTSRenderer {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	ttspb.RegisterTTSServiceServer(srv, &mockTTSServer{samples: samples})

	go func() {
		if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Logf("mock TTS server error: %v", err)
		}
	}()
	t.Cleanup(func() { srv.GracefulStop() })

	//nolint:staticcheck
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return &GRPCTTSRenderer{
		client: ttspb.NewTTSServiceClient(conn),
		conn:   conn,
	}
}

func TestGRPCTTSRenderer_Render_WritesWAV(t *testing.T) {
	// Known samples: a simple sine-like pattern.
	samples := []float32{0.0, 0.1, 0.2, -0.1, -0.2, 0.0}

	renderer := startMockTTSServer(t, samples)

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "output.wav")

	err := renderer.Render(context.Background(), "Hello world", "af_heart", outPath)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Assert WAV file was created and is non-empty.
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output WAV not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output WAV is empty")
	}

	// WAV header is 44 bytes; data section should be len(samples)*2 bytes (int16 PCM).
	expectedSize := int64(44 + len(samples)*2)
	if info.Size() != expectedSize {
		t.Errorf("WAV size: got %d bytes, want %d", info.Size(), expectedSize)
	}
}

func TestDecodePCM_RoundTrip(t *testing.T) {
	original := []float32{0.5, -0.5, 1.0, -1.0, 0.0}

	// Encode.
	pcm := make([]byte, len(original)*4)
	for i, s := range original {
		binary.LittleEndian.PutUint32(pcm[i*4:], math.Float32bits(s))
	}

	// Decode.
	decoded, err := decodePCM(pcm)
	if err != nil {
		t.Fatalf("decodePCM failed: %v", err)
	}

	if len(decoded) != len(original) {
		t.Fatalf("length mismatch: got %d, want %d", len(decoded), len(original))
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("sample[%d]: got %f, want %f", i, decoded[i], original[i])
		}
	}
}

func TestDecodePCM_InvalidLength(t *testing.T) {
	_, err := decodePCM([]byte{0x01, 0x02, 0x03}) // 3 bytes — not multiple of 4
	if err == nil {
		t.Error("expected error for odd-length PCM, got nil")
	}
}
