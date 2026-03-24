//go:build linux

package mssmodule

import (
	"fmt"
	"net"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

func getMSS(tc *net.TCPConn, logger *zap.Logger) (int, error) {
	var mss int
	var retErr error

	raw, err := tc.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("syscall conn failed: %w", err)
	}

	controlErr := raw.Control(func(fd uintptr) {
		info, err := unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
		if err != nil {
			logger.Error("GetsockoptTCPInfo failed", zap.Error(err))
			retErr = fmt.Errorf("GetsockoptTCPInfo failed: %w", err)
			return
		}
		mss = int(info.Snd_mss)

		logger.Info("TCP_INFO extracted",
			zap.Uint8("state", info.State),
			zap.Uint32("snd_mss", info.Snd_mss),
			zap.Uint32("rcv_mss", info.Rcv_mss),
			zap.Uint32("advmss", info.Advmss),
		)
	})

	if controlErr != nil {
		return 0, controlErr
	}

	return mss, retErr
}
