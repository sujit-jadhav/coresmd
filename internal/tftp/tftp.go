// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package tftp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pin/tftp/v3"
	"github.com/sirupsen/logrus"
)

const (
	DefaultTFTPDirectory = "/tftpboot"
	DefaultTFTPPort      = 69

	defaultScriptName = "default"
	defaultScript     = `#!ipxe
reboot
`
)

type ScriptReader struct{}

func (sr ScriptReader) Read(b []byte) (int, error) {
	nBytes := copy(b, []byte(defaultScript))
	return nBytes, io.EOF
}

// TftpServer configures and starts a TFTP server.
type TftpServer struct {
	Address    string
	Directory  string
	Port       int
	SinglePort bool
	Logger     *logrus.Entry
}

// Start creates, configures, and starts the TFTP server implementation.
func (t *TftpServer) Start() {
	if t.Logger == nil {
		t.Logger = logrus.NewEntry(logrus.New())
	}
	s := tftp.NewServer(readHandler(t.Logger, t.Directory), nil)
	if t.SinglePort {
		s.EnableSinglePort()
	}
	err := s.ListenAndServe(fmt.Sprintf("%s:%d", t.Address, t.Port))
	if err != nil {
		t.Logger.Fatalf("failed to start TFTP server: %v", err)
	}
}

func readHandler(logger *logrus.Entry, directory string) func(string, io.ReaderFrom) error {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	return func(filename string, rf io.ReaderFrom) error {
		raddr := "(unknown)"
		ot, ok := rf.(tftp.OutgoingTransfer)
		if !ok {
			logger.Error("unable to get remote address")
		} else {
			ra := ot.RemoteAddr()
			raptr := &ra
			if raptr != nil && raptr.IP != nil {
				raddr = raptr.IP.String()
			} else {
				return errors.New("remote address was nil, cannot send response")
			}
		}
		if filename == defaultScriptName {
			logger.Infof("tftp: %s requested default script", raddr)
			var sr ScriptReader
			nbytes, err := rf.ReadFrom(sr)
			logger.Infof("tftp: sent %d bytes of default script to %s", nbytes, raddr)
			return err
		}
		logger.Infof("tftp: %s requested file %s", raddr, filename)
		filePath := filepath.Join(directory, filename)
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		nbytes, err := rf.ReadFrom(file)
		logger.Infof("tftp: sent %d bytes of file %s to %s", nbytes, filename, raddr)
		return err
	}
}
