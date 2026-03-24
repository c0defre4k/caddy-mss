//go:build darwin

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
		mss, retErr = unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG)
		if retErr != nil {
			logger.Error("GetsockoptInt failed", zap.Error(retErr))
		} else {
			logger.Info("TCP_MAXSEG extracted", zap.Int("mss", mss))
		}
	})

	if controlErr != nil {
		return 0, controlErr
	}

	return mss, retErr
}
