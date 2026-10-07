package live

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/coder/websocket"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"strings"
	"time"
)

type Repository interface {
	ClaimLive(context.Context, string, string) (d.Session, error)
	RenewLive(context.Context, d.Session) error
	ReleaseLive(context.Context, d.Session) error
	SaveLive(context.Context, d.Session, int32, []byte, string, bool, string) error
	Queue(context.Context, string, string) (d.Session, error)
	CreateLive(context.Context, string, string, string, d.Examiner, string, []d.Turn) (d.Session, error)
}
type Provider interface {
	ConnectLive(context.Context) (*websocket.Conn, error)
}
type Service struct {
	Repo     Repository
	Provider Provider
	Model    string
}
type packet struct {
	kind websocket.MessageType
	data []byte
	err  error
}

func read(ctx context.Context, c *websocket.Conn, ch chan<- packet) {
	for {
		kind, b, e := c.Read(ctx)
		select {
		case ch <- packet{kind, b, e}:
		case <-ctx.Done():
			return
		}
		if e != nil {
			return
		}
	}
}
func write(ctx context.Context, c *websocket.Conn, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, b)
}

// Run owns progression. Provider output cannot advance the persisted interview
// without a validated tool call and actual recorded speech.
func (s *Service) Run(parent context.Context, client *websocket.Conn, session d.Session) error {
	ctx, cancel := context.WithTimeout(parent, 20*time.Minute)
	defer cancel()
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = s.Repo.ReleaseLive(c, session)
	}()
	upstream, e := s.Provider.ConnectLive(ctx)
	if e != nil {
		return e
	}
	defer upstream.CloseNow()
	pos := 0
	for pos < len(session.Turns) && session.Turns[pos].Answered {
		pos++
	}
	if pos == len(session.Turns) {
		_, e = s.Repo.Queue(ctx, session.UserID, session.ID)
		return e
	}
	if e = write(ctx, upstream, Setup(session, pos)); e != nil {
		return e
	}
	client.SetReadLimit(8192)
	incoming, provider := make(chan packet, 8), make(chan packet, 8)
	go read(ctx, client, incoming)
	go read(ctx, upstream, provider)
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	phase := "connecting"
	allowOutput := true
	outputBytes, outputRetries := 0, 0
	awaitPlayback := false
	prepared := false
	finished := false
	var prepUntil time.Time
	var pcm []byte
	transcript := ""
	helpStart := 0
	helpSpeech := ""
	audioStarted := time.Now()
	received := 0
	lastControl := time.Time{}
	spokenQuestion := func() string {
		if helpSpeech != "" {
			return helpSpeech
		}
		if finished {
			return "Thank you. That completes your interview. Your report will be prepared shortly."
		}
		if session.Turns[pos].Part == 2 && prepared {
			return "Your preparation time is over. Please begin your two-minute answer now."
		}
		return session.Turns[pos].Prompt
	}
	state := func() error {
		return write(ctx, client, map[string]any{"type": "state", "phase": phase, "position": pos, "part": session.Turns[pos].Part, "question": session.Turns[pos].Prompt, "prepare_until": prepUntil.UnixMilli()})
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-heartbeat.C:
			if e = s.Repo.RenewLive(ctx, session); e != nil {
				return e
			}
			if e = write(ctx, client, map[string]string{"type": "heartbeat"}); e != nil {
				return e
			}
		case <-tick.C:
			if phase == "preparing" && !time.Now().Before(prepUntil) {
				phase = "examiner"
				allowOutput = true
				outputBytes, outputRetries = 0, 0
				prepared = true
				prepUntil = time.Time{}
				if e = state(); e != nil {
					return e
				}
				if e = write(ctx, upstream, Control("Preparation ended. Invite the student to begin their two-minute long turn now; do not ask a new question.")); e != nil {
					return e
				}
			}
		case p := <-incoming:
			if p.err != nil {
				return p.err
			}
			if p.kind == websocket.MessageBinary {
				if phase != "listening" {
					continue
				}
				if len(p.data)%2 != 0 {
					return d.ErrInvalid
				}
				received += len(p.data)
				if received > int(time.Since(audioStarted).Seconds()*48000)+64000 {
					return d.ErrLimit
				}
				// A hard question budget bounds spend and ends the timed long turn.
				// Never manufacture an answer or score when no speech was captured.
				if len(pcm)+len(p.data) > int(session.Turns[pos].Seconds)*32000 {
					if !Meaningful(transcript[helpStart:]) {
						return d.ErrInvalid
					}
					next := ""
					if pos+1 < len(session.Turns) {
						next = session.Turns[pos+1].Prompt
					}
					if e = s.Repo.SaveLive(ctx, session, int32(pos), Wave(pcm), transcript, true, next); e != nil {
						return e
					}
					if next == "" {
						if _, e = s.Repo.Queue(ctx, session.UserID, session.ID); e != nil {
							return e
						}
						_ = write(ctx, client, map[string]string{"type": "done"})
						return nil
					}
					pos++
					pcm = nil
					transcript = ""
					helpStart = 0
					prepared = false
					allowOutput = true
					outputBytes, outputRetries = 0, 0
					phase = "examiner"
					if e = state(); e != nil {
						return e
					}
					if e = write(ctx, upstream, Control("The time for that answer has ended. Thank the student briefly and ask the next question at position "+jsonNumber(pos)+": "+next)); e != nil {
						return e
					}
					continue
				}
				pcm = append(pcm, p.data...)
				e = write(ctx, upstream, map[string]any{"realtimeInput": map[string]any{"audio": map[string]string{"data": base64.StdEncoding.EncodeToString(p.data), "mimeType": "audio/pcm;rate=16000"}}})
				if e != nil {
					return e
				}
				continue
			}
			var command struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(p.data, &command) != nil {
				return d.ErrInvalid
			}
			switch command.Type {
			case "playback_done":
				if !awaitPlayback {
					continue
				}
				awaitPlayback = false
				if finished {
					_, e = s.Repo.Queue(ctx, session.UserID, session.ID)
					if e != nil {
						return e
					}
					_ = write(ctx, client, map[string]string{"type": "done"})
					return nil
				}
				allowOutput = false
				helpSpeech = ""
				if session.Turns[pos].Part == 2 && !prepared {
					phase = "preparing"
					prepUntil = time.Now().Add(time.Minute)
				} else {
					phase = "listening"
					audioStarted = time.Now()
					received = 0
				}
				if e = state(); e != nil {
					return e
				}
			case "repeat":
				if phase != "listening" || time.Since(lastControl) < 5*time.Second {
					continue
				}
				lastControl = time.Now()
				allowOutput = true
				outputBytes, outputRetries = 0, 0
				phase = "examiner"
				_ = state()
				if e = write(ctx, upstream, Control("Repeat the current question without advancing: "+session.Turns[pos].Prompt)); e != nil {
					return e
				}
			case "done_speaking":
				if phase != "listening" || time.Since(lastControl) < 5*time.Second {
					continue
				}
				lastControl = time.Now()
				if e = write(ctx, upstream, map[string]any{"realtimeInput": map[string]bool{"audioStreamEnd": true}}); e != nil {
					return e
				}
			default:
				return d.ErrInvalid
			}
		case p := <-provider:
			if p.err != nil {
				return d.ErrUnavailable
			}
			var msg struct {
				SetupComplete json.RawMessage `json:"setupComplete"`
				GoAway        json.RawMessage `json:"goAway"`
				ServerContent struct {
					ModelTurn struct {
						Parts []struct {
							InlineData *struct {
								Data     string `json:"data"`
								MimeType string `json:"mimeType"`
							} `json:"inlineData"`
						}
					} `json:"modelTurn"`
					InputTranscription struct {
						Text string `json:"text"`
					} `json:"inputTranscription"`
					TurnComplete bool `json:"turnComplete"`
					Interrupted  bool `json:"interrupted"`
				} `json:"serverContent"`
				ToolCall struct {
					FunctionCalls []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
						Args struct {
							Question    string `json:"question"`
							Explanation string `json:"explanation"`
						} `json:"args"`
					}
				} `json:"toolCall"`
			}
			if json.Unmarshal(p.data, &msg) != nil {
				return d.ErrUnavailable
			}
			if len(msg.GoAway) > 0 {
				return d.ErrUnavailable
			}
			if len(msg.SetupComplete) > 0 {
				phase = "examiner"
				if e = state(); e != nil {
					return e
				}
				if e = write(ctx, upstream, Control("Begin or resume at position "+jsonNumber(pos)+". Ask this question aloud now: "+session.Turns[pos].Prompt)); e != nil {
					return e
				}
			}
			content := msg.ServerContent
			if phase == "listening" {
				transcript += content.InputTranscription.Text
				if len(transcript) > 16000 {
					return d.ErrLimit
				}
			}
			for _, call := range msg.ToolCall.FunctionCalls {
				response := map[string]any{"instruction": "Stay silent and wait for the student to finish."}
				if phase == "examiner" {
					response = map[string]any{"question": spokenQuestion(), "instruction": "The controller has already approved this examiner turn. Speak the supplied words aloud now; do not call another tool or wait for a student answer before asking the question."}
				}
				if phase == "listening" {
					switch call.Name {
					case "repeat_question", "clarify_question":
						// Help requests are conversational turns, never scored answers
						// or permission to advance based on an earlier transcript.
						if strings.TrimSpace(transcript[helpStart:]) == "" {
							break
						}
						if _, err := d.ValidateAudio(Wave(pcm), session.Turns[pos].Seconds); err != nil {
							break
						}
						helpSpeech = session.Turns[pos].Prompt
						if call.Name == "clarify_question" {
							explanation := strings.TrimSpace(call.Args.Explanation)
							if len(explanation) < 5 || len(explanation) > 700 {
								break
							}
							helpSpeech = explanation
						}
						helpStart = len(transcript)
						allowOutput = true
						outputBytes, outputRetries = 0, 0
						phase = "examiner"
						response = map[string]any{"question": helpSpeech, "instruction": "Say this aloud, then listen for the answer to the SAME question. Do not advance, give examples, suggest an answer, or evaluate the student."}
					case "advance_question", "finish_interview":
						valid := Meaningful(transcript[helpStart:])
						if _, err := d.ValidateAudio(Wave(pcm), session.Turns[pos].Seconds); err != nil {
							valid = false
						}
						final := pos == len(session.Turns)-1
						if valid && ((final && call.Name == "finish_interview") || (!final && call.Name == "advance_question")) {
							next := ""
							if !final {
								next = strings.TrimSpace(call.Args.Question)
								if session.Turns[pos+1].Part == 2 {
									next = session.Turns[pos+1].Prompt
								}
								if len(next) < 8 || len(next) > 1600 {
									break
								}
							}
							if e = s.Repo.SaveLive(ctx, session, int32(pos), Wave(pcm), transcript, true, next); e != nil {
								return e
							}
							session.Turns[pos].Answered = true
							pcm = nil
							transcript = ""
							helpStart = 0
							allowOutput = true
							outputBytes, outputRetries = 0, 0
							phase = "examiner"
							if final {
								finished = true
								response = map[string]any{"instruction": "The interview is complete. Thank the student briefly and say their report will be prepared."}
							} else {
								pos++
								session.Turns[pos].Prompt = next
								prepared = false
								response = map[string]any{"position": pos, "question": next, "instruction": "Ask this approved question aloud. Then listen."}
							}
						}
					}
				}
				if e = write(ctx, upstream, map[string]any{"toolResponse": map[string]any{"functionResponses": []any{map[string]any{"id": call.ID, "name": call.Name, "response": response}}}}); e != nil {
					return e
				}
				if e = state(); e != nil {
					return e
				}
			}
			if content.Interrupted {
				if e = write(ctx, client, map[string]string{"type": "interrupted"}); e != nil {
					return e
				}
			}
			if allowOutput {
				for _, part := range content.ModelTurn.Parts {
					if part.InlineData != nil {
						b, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
						if err != nil {
							return d.ErrUnavailable
						}
						if !strings.HasPrefix(part.InlineData.MimeType, "audio/pcm") {
							return d.ErrUnavailable
						}
						outputBytes += len(b)
						c, stop := context.WithTimeout(ctx, 10*time.Second)
						e = client.Write(c, websocket.MessageBinary, b)
						stop()
						if e != nil {
							return e
						}
					}
				}
			}
			if content.TurnComplete && allowOutput {
				// Some provider turns contain only tool calls. Never tell the
				// browser to start listening when no question was actually spoken.
				if outputBytes < 4800 {
					if outputRetries >= 2 {
						return d.ErrUnavailable
					}
					outputRetries++
					if e = write(ctx, upstream, Control("No audible examiner response was received. Do not call tools. Speak these approved words aloud now: "+spokenQuestion())); e != nil {
						return e
					}
					continue
				}
				awaitPlayback = true
				if e = write(ctx, client, map[string]string{"type": "audio_end"}); e != nil {
					return e
				}
			}
		}
	}
}
func jsonNumber(n int) string { b, _ := json.Marshal(n); return string(b) }
