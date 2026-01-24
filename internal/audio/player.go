package audio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

const playbackPollInterval = 100 * time.Millisecond

// StreamToSpeakers decodes MP3 audio from the reader and plays it to the default output device.
func StreamToSpeakers(ctx context.Context, r io.Reader) error {
	decoder, err := mp3.NewDecoder(r)
	if err != nil {
		return fmt.Errorf("decode mp3: %w", err)
	}

	const (
		channelCount = 2
		format       = oto.FormatSignedInt16LE
	)

	audioCtx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   decoder.SampleRate(),
		ChannelCount: channelCount,
		Format:       format,
	})
	if err != nil {
		return fmt.Errorf("audio context: %w", err)
	}
	<-ready

	player := audioCtx.NewPlayer(decoder)
	player.Play()

	return waitForPlayback(ctx, player)
}

func waitForPlayback(ctx context.Context, player *oto.Player) error {
	ticker := time.NewTicker(playbackPollInterval)
	defer ticker.Stop()

	for {
		if !player.IsPlaying() {
			return nil
		}
		select {
		case <-ctx.Done():
			player.Pause()
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
