//go:build linux

// chiron-ink is the AppLoad backend that gives pen strokes native-feeling
// latency on the reMarkable. The QML layer cannot get the e-ink fast
// waveform - xochitl reserves it for surfaces it knows - so ink goes
// through AppLoad's qtfb channel instead: this process owns a transparent
// RGBA framebuffer (displayed by an FBController in the frontend), receives
// pen events directly, stamps stroke segments into shared memory, and
// requests segment-sized partial refreshes in UFAST mode.
//
// The frontend stays the source of truth for submission: it pushes the
// current page's zones and stored strokes on every page change, and pulls
// everything back with a flush before check-ins. Stroke coordinates cross
// the wire normalized to the page, exactly like the QML canvas kept them.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

func unsafePointer(v *[6]int32) unsafe.Pointer { return unsafe.Pointer(v) }

const (
	fbW = 1620
	fbH = 2160
	// The rendezvous key shared with the frontend's FBController.
	fbKey = 0x43484952 // "CHIR"

	qtfbSocket = "/tmp/qtfb.sock"

	msgInitialize     = 0
	msgUpdate         = 1
	msgTerminate      = 3
	msgUserInput      = 4
	msgSetRefreshMode = 5

	fbfmtRMPPRGBA8888 = 2
	updatePartial     = 1
	refreshUFast      = 0

	inputPenPress   = 0x20
	inputPenRelease = 0x21
	inputPenUpdate  = 0x22

	// Frontend -> backend
	mState = 1
	mFlush = 2
	// Backend -> frontend
	mReady   = 100
	mStrokes = 101

	nib = 3 // stroke radius, px
)

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type state struct {
	Enabled bool      `json:"enabled"`
	Page    int       `json:"page"`
	Zones   [][4]int  `json:"zones"` // page px; empty means whole page
	Strokes [][]point `json:"strokes"`
}

type ink struct {
	mu      sync.Mutex
	fb      []byte
	qtfbFD  int
	enabled bool
	page    int
	zones   [][4]int
	strokes [][]point
	live    []point
	lastX   int
	lastY   int
	down    bool

	// Dirty-rect accumulator: pixels land in the framebuffer immediately,
	// but refresh requests are coalesced to ~66Hz - the display engine
	// coalesces in-flight updates fine, xochitl's socket + event loop do
	// not (research brief), and 500 messages/sec falls behind.
	dirty      [4]int
	dirtyValid bool

	// Latency forensics: where does the time go between pen events?
	strokeStart time.Time
	lastEvent   time.Time
	gapSum      time.Duration
	gapMax      time.Duration
	procSum     time.Duration
	procMax     time.Duration
	events      int
}

func seqConnect(path string) (int, error) {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		return -1, err
	}
	if err := syscall.Connect(fd, &syscall.SockaddrUnix{Name: path}); err != nil {
		syscall.Close(fd)
		return -1, err
	}
	return fd, nil
}

// qtfb wire structs, hand-packed to the C ABI (LP64, little-endian).
func qtfbInit(fd int) ([]byte, error) {
	msg := make([]byte, 24)
	msg[0] = msgInitialize
	binary.LittleEndian.PutUint32(msg[4:], uint32(fbKey))
	msg[8] = fbfmtRMPPRGBA8888
	if _, err := syscall.Write(fd, msg); err != nil {
		return nil, err
	}
	resp := make([]byte, 32)
	n, err := syscall.Read(fd, resp)
	if err != nil || n < 24 {
		return nil, fmt.Errorf("init response: n=%d err=%v", n, err)
	}
	shmKey := int32(binary.LittleEndian.Uint32(resp[8:]))
	shmSize := binary.LittleEndian.Uint64(resp[16:])
	shmFD, err := syscall.Open(fmt.Sprintf("/dev/shm/qtfb_%d", shmKey), syscall.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("shm open: %w", err)
	}
	mem, err := syscall.Mmap(shmFD, 0, int(shmSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap: %w", err)
	}
	return mem, nil
}

func qtfbSend24(fd int, b []byte) {
	if _, err := syscall.Write(fd, b); err != nil {
		log.Printf("qtfb write: %v", err)
	}
}

func setRefreshMode(fd, mode int) {
	msg := make([]byte, 24)
	msg[0] = msgSetRefreshMode
	binary.LittleEndian.PutUint32(msg[4:], uint32(mode))
	qtfbSend24(fd, msg)
}

func partialUpdate(fd, x, y, w, h int) {
	msg := make([]byte, 24)
	msg[0] = msgUpdate
	binary.LittleEndian.PutUint32(msg[4:], updatePartial)
	binary.LittleEndian.PutUint32(msg[8:], uint32(x))
	binary.LittleEndian.PutUint32(msg[12:], uint32(y))
	binary.LittleEndian.PutUint32(msg[16:], uint32(w))
	binary.LittleEndian.PutUint32(msg[20:], uint32(h))
	qtfbSend24(fd, msg)
}

// markDirty grows the pending refresh rect; flushDirty sends it.
func (k *ink) markDirty(x0, y0, x1, y1 int) {
	if !k.dirtyValid {
		k.dirty = [4]int{x0, y0, x1, y1}
		k.dirtyValid = true
		return
	}
	k.dirty[0] = min(k.dirty[0], x0)
	k.dirty[1] = min(k.dirty[1], y0)
	k.dirty[2] = max(k.dirty[2], x1)
	k.dirty[3] = max(k.dirty[3], y1)
}

func (k *ink) flushDirtyLocked() {
	if !k.dirtyValid {
		return
	}
	partialUpdate(k.qtfbFD, k.dirty[0], k.dirty[1],
		k.dirty[2]-k.dirty[0]+1, k.dirty[3]-k.dirty[1]+1)
	k.dirtyValid = false
}

func (k *ink) inZone(x, y int) bool {
	if len(k.zones) == 0 {
		return true
	}
	for _, z := range k.zones {
		if x >= z[0] && x <= z[0]+z[2] && y >= z[1] && y <= z[1]+z[3] {
			return true
		}
	}
	return false
}

func (k *ink) stamp(x, y int) {
	for dy := -nib; dy <= nib; dy++ {
		for dx := -nib; dx <= nib; dx++ {
			if dx*dx+dy*dy > nib*nib {
				continue
			}
			px, py := x+dx, y+dy
			if px < 0 || px >= fbW || py < 0 || py >= fbH {
				continue
			}
			o := (py*fbW + px) * 4
			k.fb[o] = 0
			k.fb[o+1] = 0
			k.fb[o+2] = 0
			k.fb[o+3] = 255
		}
	}
}

func (k *ink) segment(x0, y0, x1, y1 int) {
	dx, dy := x1-x0, y1-y0
	steps := max(abs(dx), abs(dy))
	if steps == 0 {
		k.stamp(x0, y0)
	} else {
		for i := 0; i <= steps; i++ {
			k.stamp(x0+dx*i/steps, y0+dy*i/steps)
		}
	}
}

func (k *ink) clear() {
	for i := range k.fb {
		k.fb[i] = 0
	}
	partialUpdate(k.qtfbFD, 0, 0, fbW, fbH)
}

func (k *ink) redraw() {
	for i := range k.fb {
		k.fb[i] = 0
	}
	for _, s := range k.strokes {
		for i := 1; i < len(s); i++ {
			k.segment(int(s[i-1].X*fbW), int(s[i-1].Y*fbH),
				int(s[i].X*fbW), int(s[i].Y*fbH))
		}
		if len(s) == 1 {
			k.stamp(int(s[0].X*fbW), int(s[0].Y*fbH))
		}
	}
	partialUpdate(k.qtfbFD, 0, 0, fbW, fbH)
}

func (k *ink) pen(kind, x, y, d int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.enabled {
		return
	}
	switch kind {
	case inputPenPress:
		if !k.inZone(x, y) {
			return
		}
		now := time.Now()
		k.down = true
		k.live = []point{{float64(x) / fbW, float64(y) / fbH}}
		k.stamp(x, y)
		// First contact refreshes immediately - the dot must not wait a tick.
		partialUpdate(k.qtfbFD, x-nib, y-nib, 2*nib+1, 2*nib+1)
		k.lastX, k.lastY = x, y
		k.strokeStart = now
		k.lastEvent = now
		k.gapSum, k.gapMax, k.procSum, k.procMax = 0, 0, 0, 0
		k.events = 0
	case inputPenUpdate:
		if !k.down {
			return
		}
		if !k.inZone(x, y) {
			k.endStrokeLocked()
			return
		}
		now := time.Now()
		gap := now.Sub(k.lastEvent)
		k.lastEvent = now
		k.gapSum += gap
		if gap > k.gapMax {
			k.gapMax = gap
		}
		k.events++
		k.segment(k.lastX, k.lastY, x, y)
		k.markDirty(min(k.lastX, x)-nib, min(k.lastY, y)-nib,
			max(k.lastX, x)+nib, max(k.lastY, y)+nib)
		proc := time.Since(now)
		k.procSum += proc
		if proc > k.procMax {
			k.procMax = proc
		}
		k.live = append(k.live, point{float64(x) / fbW, float64(y) / fbH})
		k.lastX, k.lastY = x, y
	case inputPenRelease:
		k.flushDirtyLocked()
		k.endStrokeLocked()
	}
}

// refreshLoop drains the dirty rect at ~66Hz while the pen is down.
func (k *ink) refreshLoop() {
	t := time.NewTicker(15 * time.Millisecond)
	for range t.C {
		k.mu.Lock()
		k.flushDirtyLocked()
		k.mu.Unlock()
	}
}

func (k *ink) endStrokeLocked() {
	if k.down && len(k.live) > 1 {
		k.strokes = append(k.strokes, k.live)
		if k.events > 0 {
			// One line per stroke: if event gaps are large, latency is
			// upstream of us (input delivery); if proc is large, it is
			// ours; if both are small and ink still lags, it is the
			// refresh pipeline downstream.
			log.Printf("stroke: %d pts in %dms | evt gap avg %.1fms max %.1fms | proc avg %dus max %dus",
				len(k.live), time.Since(k.strokeStart).Milliseconds(),
				float64(k.gapSum.Microseconds())/float64(k.events)/1000,
				float64(k.gapMax.Microseconds())/1000,
				k.procSum.Microseconds()/int64(k.events), k.procMax.Microseconds())
		}
	}
	k.down = false
	k.live = nil
}

func (k *ink) setState(s state) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.enabled = s.Enabled
	k.page = s.Page
	k.zones = s.Zones
	k.strokes = s.Strokes
	if k.strokes == nil {
		k.strokes = [][]point{}
	}
	k.down = false
	k.live = nil
	if s.Enabled {
		k.redraw()
	} else {
		k.clear()
	}
}

func (k *ink) flush() (int, [][]point) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.endStrokeLocked()
	out := make([][]point, len(k.strokes))
	copy(out, k.strokes)
	return k.page, out
}

func sendAppLoad(fd int, msgType uint32, contents string) {
	head := make([]byte, 8)
	binary.LittleEndian.PutUint32(head, msgType)
	binary.LittleEndian.PutUint32(head[4:], uint32(len(contents)))
	if _, err := syscall.Write(fd, head); err != nil {
		log.Printf("appload write: %v", err)
		return
	}
	if len(contents) > 0 {
		if _, err := syscall.Write(fd, []byte(contents)); err != nil {
			log.Printf("appload write body: %v", err)
		}
	}
}

// evdevPen reads the pen digitizer directly - the Qt event path adds a
// ~130ms stall at every stroke start plus buffering (measured), so the
// hardware stream is the pen source. Calibrated against Qt-forwarded
// events: the digitizer maps straight onto the portrait screen,
// screen = raw * (1620/11180, 2160/15340), no swap, no flip.
func evdevPen(k *ink) {
	fd, err := syscall.Open("/dev/input/event2", syscall.O_RDONLY, 0)
	if err != nil {
		log.Printf("evdev open: %v", err)
		return
	}
	absinfo := func(axis int) (int32, int32) {
		var info [6]int32
		req := uintptr(0x80184540 + axis)
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req,
			uintptr(unsafePointer(&info)))
		if errno != 0 {
			return 0, 0
		}
		return info[1], info[2] // min, max
	}
	x0, x1 := absinfo(0)
	y0, y1 := absinfo(1)
	p0, p1 := absinfo(24)
	log.Printf("evdev pen ranges: x[%d..%d] y[%d..%d] p[%d..%d]", x0, x1, y0, y1, p0, p1)

	if x1 == 0 || y1 == 0 {
		x1, y1 = 11180, 15340 // measured Paper Pro ranges
	}
	_ = p0
	_ = p1
	scaleX := float64(fbW) / float64(x1)
	scaleY := float64(fbH) / float64(y1)

	buf := make([]byte, 24*64)
	var x, y int32
	touching := false
	for {
		n, err := syscall.Read(fd, buf)
		if err != nil || n == 0 {
			log.Printf("evdev read: n=%d err=%v", n, err)
			return
		}
		for o := 0; o+24 <= n; o += 24 {
			typ := binary.LittleEndian.Uint16(buf[o+16:])
			code := binary.LittleEndian.Uint16(buf[o+18:])
			val := int32(binary.LittleEndian.Uint32(buf[o+20:]))
			switch typ {
			case 1: // EV_KEY
				if code == 330 { // BTN_TOUCH
					was := touching
					touching = val != 0
					sx, sy := int(float64(x)*scaleX), int(float64(y)*scaleY)
					if touching && !was {
						k.pen(inputPenPress, sx, sy, 0)
					} else if !touching && was {
						k.pen(inputPenRelease, sx, sy, 0)
					}
				}
			case 3: // EV_ABS
				switch code {
				case 0:
					x = val
				case 1:
					y = val
				}
			case 0: // SYN_REPORT
				if touching {
					k.pen(inputPenUpdate, int(float64(x)*scaleX), int(float64(y)*scaleY), 0)
				}
			}
		}
	}
}

func main() {
	log.SetPrefix("[chiron-ink] ")
	if len(os.Args) < 2 {
		log.Fatal("usage: chiron-ink <appload-socket>")
	}
	alFD, err := seqConnect(os.Args[1])
	if err != nil {
		log.Fatalf("appload socket: %v", err)
	}
	qfd, err := seqConnect(qtfbSocket)
	if err != nil {
		log.Fatalf("qtfb socket: %v", err)
	}
	fb, err := qtfbInit(qfd)
	if err != nil {
		log.Fatalf("qtfb init: %v", err)
	}
	k := &ink{fb: fb, qtfbFD: qfd}
	setRefreshMode(qfd, refreshUFast)
	log.Printf("up: fb %d bytes, ufast", len(fb))
	go evdevPen(k)
	go k.refreshLoop()

	// qtfb reader: pen input arrives here.
	go func() {
		buf := make([]byte, 32)
		for {
			n, err := syscall.Read(qfd, buf)
			if err != nil || n == 0 {
				log.Printf("qtfb closed: n=%d err=%v", n, err)
				os.Exit(0)
			}
			if buf[0] != msgUserInput || n < 28 {
				continue
			}
			kind := int(int32(binary.LittleEndian.Uint32(buf[8:])))
			x := int(int32(binary.LittleEndian.Uint32(buf[16:])))
			y := int(int32(binary.LittleEndian.Uint32(buf[20:])))
			d := int(int32(binary.LittleEndian.Uint32(buf[24:])))
			// Pen input comes from evdev now; Qt-forwarded events are
			// drained and dropped (they trail the hardware by ~100ms).
			_ = kind
			_, _, _ = x, y, d
		}
	}()

	sendAppLoad(alFD, mReady, "")

	// AppLoad reader: frontend control messages.
	head := make([]byte, 8)
	body := make([]byte, 1<<20)
	for {
		n, err := syscall.Read(alFD, head)
		if err != nil || n == 0 {
			log.Printf("appload closed: %v", err)
			return
		}
		if n < 8 {
			continue
		}
		msgType := binary.LittleEndian.Uint32(head)
		length := binary.LittleEndian.Uint32(head[4:])
		payload := ""
		if length > 0 {
			bn, err := syscall.Read(alFD, body[:length])
			if err != nil {
				log.Printf("appload body: %v", err)
				return
			}
			payload = string(body[:bn])
		}
		switch msgType {
		case 0xFFFFFFFF: // terminate
			return
		case 0xFFFFFFFE: // new coordinator: re-announce
			sendAppLoad(alFD, mReady, "")
		case mState:
			var s state
			if err := json.Unmarshal([]byte(payload), &s); err == nil {
				k.setState(s)
			}
		case mFlush:
			page, strokes := k.flush()
			out, _ := json.Marshal(map[string]any{"page": page, "strokes": strokes})
			sendAppLoad(alFD, mStrokes, string(out))
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
