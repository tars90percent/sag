package minimax

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const defaultBaseURL = "https://api.minimax.io"

// Client talks to the MiniMax TTS API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient returns a client configured with the given API key and base URL.
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Voice represents a MiniMax voice entry.
type Voice struct {
	VoiceID     string
	Name        string
	Category    string
	Description string
}

type voiceEntry struct {
	VoiceID     string   `json:"voice_id"`
	VoiceName   string   `json:"voice_name"`
	Description []string `json:"description,omitempty"`
}

type listVoicesRequest struct {
	VoiceType string `json:"voice_type"`
}

type listVoicesResponse struct {
	SystemVoice     []voiceEntry `json:"system_voice"`
	VoiceCloning    []voiceEntry `json:"voice_cloning"`
	VoiceGeneration []voiceEntry `json:"voice_generation"`
	BaseResp        *baseResp    `json:"base_resp,omitempty"`
}

// ListVoices fetches available voices.
func (c *Client) ListVoices(ctx context.Context) ([]Voice, error) {
	u, err := c.httpURL("/v1/get_voice")
	if err != nil {
		return nil, err
	}

	reqBody, err := json.Marshal(listVoicesRequest{VoiceType: "all"})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list voices failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload listVoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if err := payload.BaseResp.err(); err != nil {
		return nil, err
	}

	voices := make([]Voice, 0, len(payload.SystemVoice)+len(payload.VoiceCloning)+len(payload.VoiceGeneration))
	appendVoices := func(category string, entries []voiceEntry) {
		for _, v := range entries {
			name := strings.TrimSpace(v.VoiceName)
			if name == "" {
				name = v.VoiceID
			}
			voices = append(voices, Voice{
				VoiceID:     v.VoiceID,
				Name:        name,
				Category:    category,
				Description: strings.Join(v.Description, " "),
			})
		}
	}
	appendVoices("system", payload.SystemVoice)
	appendVoices("voice_cloning", payload.VoiceCloning)
	appendVoices("voice_generation", payload.VoiceGeneration)
	return voices, nil
}

// TTSRequest configures a text-to-speech request payload.
type TTSRequest struct {
	Model           string
	Text            string
	Speed           float64
	Volume          float64
	Pitch           int
	AudioFormat     string
	SampleRate      int
	Bitrate         int
	Channel         int
	LanguageBoost   string
	ContinuousSound *bool
}

type baseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func (b *baseResp) err() error {
	if b == nil {
		return nil
	}
	if b.StatusCode == 0 {
		return nil
	}
	msg := strings.TrimSpace(b.StatusMsg)
	if msg == "" {
		msg = "unknown error"
	}
	return fmt.Errorf("minimax error: %s (code=%d)", msg, b.StatusCode)
}

type voiceSetting struct {
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
	Vol     float64 `json:"vol"`
	Pitch   int     `json:"pitch"`
}

type audioSetting struct {
	Format     string `json:"format,omitempty"`
	SampleRate int    `json:"sample_rate,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Channel    int    `json:"channel,omitempty"`
}

type t2aRequest struct {
	Model           string       `json:"model"`
	Text            string       `json:"text"`
	Stream          bool         `json:"stream"`
	OutputFormat    string       `json:"output_format,omitempty"`
	VoiceSetting    voiceSetting `json:"voice_setting"`
	AudioSetting    audioSetting `json:"audio_setting,omitempty"`
	LanguageBoost   string       `json:"language_boost,omitempty"`
	ContinuousSound *bool        `json:"continuous_sound,omitempty"`
}

type t2aResponse struct {
	Data struct {
		Audio string `json:"audio"`
	} `json:"data"`
	BaseResp *baseResp `json:"base_resp,omitempty"`
}

// ConvertTTS downloads the full audio before returning.
func (c *Client) ConvertTTS(ctx context.Context, voiceID string, req TTSRequest) ([]byte, error) {
	u, err := c.httpURL("/v1/t2a_v2")
	if err != nil {
		return nil, err
	}

	payload := t2aRequest{
		Model:           req.Model,
		Text:            req.Text,
		Stream:          false,
		OutputFormat:    "hex",
		VoiceSetting:    buildVoiceSetting(voiceID, req),
		AudioSetting:    buildAudioSetting(req),
		LanguageBoost:   req.LanguageBoost,
		ContinuousSound: req.ContinuousSound,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("convert TTS failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var response t2aResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	if err := response.BaseResp.err(); err != nil {
		return nil, err
	}
	if response.Data.Audio == "" {
		return nil, errors.New("minimax response missing audio")
	}

	data, err := hex.DecodeString(response.Data.Audio)
	if err != nil {
		return nil, fmt.Errorf("decode audio hex: %w", err)
	}
	return data, nil
}

type wsTaskStart struct {
	Event           string       `json:"event"`
	Model           string       `json:"model"`
	VoiceSetting    voiceSetting `json:"voice_setting"`
	AudioSetting    audioSetting `json:"audio_setting,omitempty"`
	LanguageBoost   string       `json:"language_boost,omitempty"`
	ContinuousSound *bool        `json:"continuous_sound,omitempty"`
}

type wsTaskContinue struct {
	Event string `json:"event"`
	Text  string `json:"text"`
}

type wsMessage struct {
	Event    string    `json:"event"`
	Data     *wsData   `json:"data,omitempty"`
	BaseResp *baseResp `json:"base_resp,omitempty"`
	IsFinal  bool      `json:"is_final,omitempty"`
}

type wsData struct {
	Audio string `json:"audio,omitempty"`
}

type cancelReadCloser struct {
	*io.PipeReader
	cancel func()
}

func (c *cancelReadCloser) Close() error {
	c.cancel()
	return c.PipeReader.Close()
}

// StreamTTS streams MP3 audio from MiniMax via WebSocket.
func (c *Client) StreamTTS(ctx context.Context, voiceID string, req TTSRequest) (io.ReadCloser, error) {
	wsURL, err := c.wsURL("/ws/v1/t2a_v2")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	pr, pw := io.Pipe()

	go func() {
		defer cancel()
		defer func() { _ = pw.Close() }()

		header := http.Header{}
		header.Set("Authorization", "Bearer "+c.apiKey)
		conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		defer func() {
			_ = conn.Close(websocket.StatusNormalClosure, "done")
		}()

		if err := readWSUntilEvent(ctx, conn, "connected_success"); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		start := wsTaskStart{
			Event:           "task_start",
			Model:           req.Model,
			VoiceSetting:    buildVoiceSetting(voiceID, req),
			AudioSetting:    buildAudioSetting(req),
			LanguageBoost:   req.LanguageBoost,
			ContinuousSound: req.ContinuousSound,
		}
		if err := wsjson.Write(ctx, conn, start); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if err := readWSUntilEvent(ctx, conn, "task_started"); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if err := wsjson.Write(ctx, conn, wsTaskContinue{Event: "task_continue", Text: req.Text}); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		for {
			var msg wsMessage
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if err := msg.BaseResp.err(); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if msg.Event == "task_failed" {
				_ = pw.CloseWithError(errors.New("minimax stream failed"))
				return
			}
			if msg.Data != nil && msg.Data.Audio != "" {
				chunk, err := hex.DecodeString(msg.Data.Audio)
				if err != nil {
					_ = pw.CloseWithError(fmt.Errorf("decode audio chunk: %w", err))
					return
				}
				if len(chunk) > 0 {
					if _, err := pw.Write(chunk); err != nil {
						return
					}
				}
			}
			if msg.IsFinal || msg.Event == "task_finished" {
				return
			}
		}
	}()

	return &cancelReadCloser{PipeReader: pr, cancel: cancel}, nil
}

func readWSUntilEvent(ctx context.Context, conn *websocket.Conn, want string) error {
	for {
		var msg wsMessage
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return err
		}
		if err := msg.BaseResp.err(); err != nil {
			return err
		}
		if msg.Event == "task_failed" {
			return errors.New("minimax task failed")
		}
		if msg.Event == want {
			return nil
		}
	}
}

func buildVoiceSetting(voiceID string, req TTSRequest) voiceSetting {
	return voiceSetting{
		VoiceID: voiceID,
		Speed:   req.Speed,
		Vol:     req.Volume,
		Pitch:   req.Pitch,
	}
}

func buildAudioSetting(req TTSRequest) audioSetting {
	return audioSetting{
		Format:     req.AudioFormat,
		SampleRate: req.SampleRate,
		Bitrate:    req.Bitrate,
		Channel:    req.Channel,
	}
}

func (c *Client) httpURL(endpoint string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, endpoint)
	return u.String(), nil
}

func (c *Client) wsURL(endpoint string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		u.Scheme = "wss"
	}
	u.Path = path.Join(u.Path, endpoint)
	return u.String(), nil
}
