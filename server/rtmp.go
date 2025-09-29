package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/flv"
	"github.com/nareix/joy4/format/rtmp"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"realtime_translate/config"
)

type RTMPServer struct {
	cfg   config.RTMPServer
	srv   *rtmp.Server
	errCh chan error
}

func NewRTMPServer(ctx context.Context, cfg config.RTMPServer) (*RTMPServer, error) {

	errCh := make(chan error, 1)
	srv := &rtmp.Server{
		Addr: cfg.Addr,
		HandlePublish: func(conn *rtmp.Conn) {
			handlePublish(ctx, conn, errCh)
		},
	}

	return &RTMPServer{
		cfg: cfg,
		srv: srv,
	}, nil
}

func (r *RTMPServer) ListenRtmp() {
	go func() {
		r.errCh <- r.srv.ListenAndServe()
	}()
}

func (r *RTMPServer) Stop() {
}

// RTMP 퍼블리시(송출) 핸들러: 오디오 스트림 코덱 정보 확인 + 오디오 패킷만 수신
func handlePublish(ctx context.Context, conn *rtmp.Conn, errCh chan error) {
	defer func() {
		_ = conn.Close()
	}()

	// 스트림(코덱) 메타데이터
	streams, err := conn.Streams()
	if err != nil {
		slog.Error("failed to read streams", "err", err)
		return
	}

	var audioOnly []av.CodecData
	for _, st := range streams {
		if st.Type().IsAudio() {
			audioOnly = append(audioOnly, st)
		}
	}
	if len(audioOnly) == 0 {
		errCh <- errors.New("no audio streams found")
		return
	}

	// ffmpeg: flv(stdin) → s16le 16k mono(stdout)
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-f", "flv",
		"-i", "pipe:0",
		"-vn",
		"-acodec", "pcm_s16le",
		"-ac", "1",
		"-ar", "16000",
		"-f", "s16le",
		"pipe:1",
	)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		errCh <- fmt.Errorf("failed to create stdin pipe: %v", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errCh <- fmt.Errorf("failed to create stdout pipe: %v", err)
		return
	}

	// FLV muxer를 ffmpeg stdin에 연결
	mux := flv.NewMuxer(stdin)
	if err := mux.WriteHeader(audioOnly); err != nil {
		errCh <- fmt.Errorf("failed to write audio only: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		errCh <- fmt.Errorf("failed to start ffmpeg: %v", err)
		return
	}

	slog.Info("ffmpeg started (stdin=flv, stdout=s16le@16kHz mono)")

	// RTMP 패킷 → FLV muxer 로 전송
	pktErrCh := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				pktErrCh <- context.Canceled
				return
			default:
			}
			pkt, err := conn.ReadPacket()
			if err != nil {
				pktErrCh <- err
				return
			}
			if err := mux.WritePacket(pkt); err != nil {
				pktErrCh <- err
				return
			}
		}
	}()

	// FFmpeg PCM 출력(20ms=640B) 읽기 고루틴
	pcmErrCh := make(chan error, 1)
	go func() {
		const chunk = 640
		r := bufio.NewReaderSize(stdout, 64*1024)
		buf := make([]byte, chunk)
		for {
			if _, err := io.ReadFull(r, buf); err != nil {
				pcmErrCh <- err
				return
			}
			if err := sendToOpenAI(buf); err != nil {
				pcmErrCh <- err
				return
			}
		}
	}()

	// 종료 처리
	var finalErr error
	select {
	case err := <-pcmErrCh:
		finalErr = err
	case err := <-pktErrCh:
		finalErr = err
	case <-ctx.Done():
		finalErr = context.Canceled
	}

	// mux/ffmpeg 정리
	_ = mux.WriteTrailer()
	_ = stdin.Close()
	_ = cmd.Wait()

	if finalErr != nil && !errors.Is(finalErr, context.Canceled) {
		slog.Error("pipeline stopped", "err", finalErr)
	} else {
		slog.Info("pipeline stopped")
	}
}

func sendToOpenAI(pcm20ms []byte) error {
	// TODO: 실제 업링크 구현
	slog.Info("sending to openAI", "len", len(pcm20ms))
	return nil
}
