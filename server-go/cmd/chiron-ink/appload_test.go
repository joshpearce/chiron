//go:build linux

package main

import (
	"encoding/binary"
	"os"
	"syscall"
	"testing"
	"time"
)

// AppLoad frames every frontend message as a header packet followed by a
// payload packet - including a ZERO-LENGTH payload packet when the QML
// side sends an empty string (the flush message does). On SOCK_SEQPACKET
// a zero-length packet reads exactly like EOF (n=0, err=nil), so the
// loop must poll to tell them apart instead of treating every zero read
// as a hangup - the first page turn used to kill the backend this way.
func TestApploadLoopSurvivesEmptyPayloadPacket(t *testing.T) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		t.Fatal(err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	k := &ink{fb: make([]byte, fbW*fbH*4), qtfbFD: int(null.Fd())}

	done := make(chan struct{})
	go func() {
		apploadLoop(fds[1], k)
		close(done)
	}()

	sendFlush := func() {
		head := make([]byte, 8)
		binary.LittleEndian.PutUint32(head, mFlush)
		if _, err := syscall.Write(fds[0], head); err != nil {
			t.Errorf("write head: %v", err)
		}
		// AppLoad's zero-length payload packet for the empty string.
		if _, err := syscall.Write(fds[0], []byte{}); err != nil {
			t.Errorf("write empty payload: %v", err)
		}
	}

	msgs := make(chan uint32, 4)
	go func() {
		buf := make([]byte, 1<<20)
		for {
			n, err := syscall.Read(fds[0], buf)
			if err != nil || n == 0 && peerHungUp(fds[0]) {
				return
			}
			if n >= 8 {
				msgs <- binary.LittleEndian.Uint32(buf)
				if length := binary.LittleEndian.Uint32(buf[4:]); length > 0 {
					syscall.Read(fds[0], buf) // drain the payload packet
				}
			}
		}
	}()
	expect := func(want uint32, what string) {
		select {
		case got := <-msgs:
			if got != want {
				t.Fatalf("%s: got message type %d, want %d", what, got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: no reply - loop treated the empty packet as a hangup", what)
		}
	}

	sendFlush()
	expect(mStrokes, "first flush")
	sendFlush()
	expect(mStrokes, "second flush")

	// Shutdown, not Close: the reader goroutine's blocked read holds the
	// file description open, so a bare Close never reaches the peer.
	syscall.Shutdown(fds[0], syscall.SHUT_RDWR)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not exit on real close")
	}
}
