package server

import (
	"context"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/rtmp"
	"log/slog"
	"realtime_translate/config"
	"time"
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

	remote := conn.NetConn().RemoteAddr().String()
	slog.Info("publisher connected", "remote", remote)

	// 스트림(코덱) 메타데이터
	streams, err := conn.Streams()
	if err != nil {
		slog.Error("failed to read streams", "err", err)
		return
	}

	// 오디오 스트림 존재 여부/코덱 정보 출력
	audioIdx := -1
	var audioCodec av.CodecData
	for i, st := range streams {
		if st.Type().IsAudio() {
			audioIdx = i
			audioCodec = st
			break
		}
	}
	if audioIdx < 0 {
		slog.Warn("no audio stream in RTMP publish")
		return
	}

	slog.Info("audio codec", "index", audioIdx, "codec", audioCodec.Type())

	// 패킷 루프: 오디오 패킷만 선별
	for {
		select {
		case <-ctx.Done():
			slog.Info("rtmp publisher stopped")
			return
		case err := <-errCh:
			slog.Error("rtmp publisher stopped with error", "err", err)
			return
		default:
		}

		pkt, err := conn.ReadPacket()
		if err != nil {
			slog.Error("read packet error", "err", err)
			break
		}

		// pkt.Data 가 압축된 원시 오디오 프레임(AAC/MP3 등)
		// pkt.Time 은 스트림 타임스탬프(상대 시간)
		slog.Info(
			"recv audio",
			"bytes", len(pkt.Data),
			"ts_ms", pkt.Time/time.Millisecond,
			"keyframe", pkt.IsKeyFrame, // 오디오에선 의미 제한적
		)
	}
	slog.Info("publisher disconnected", "remote", remote)
}
