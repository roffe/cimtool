package avr

import (
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"go.bug.st/serial"
)

//go:embed firmware.hex
var firmwareHex []byte

// STK500v1 protocol constants (see optiboot / stk500.h)
const (
	stkOK        = 0x10
	stkInsync    = 0x14
	crcEOP       = 0x20
	cmdGetSync   = 0x30
	cmdEnterPM   = 0x50
	cmdLeavePM   = 0x51
	cmdChipErase = 0x52
	cmdLoadAddr  = 0x55
	cmdProgPage  = 0x64
	cmdReadSign  = 0x75

	pageSize  = 128   // ATmega328P/PB flash page in bytes
	flashSize = 32768 // ATmega328P/PB flash in bytes
)

// urprotocol (urboot bootloaders, avrdude -c urclock) constants, see avrdude urclock.c
const (
	urProgPageFL = 0x02
	urReadPageFL = 0x03

	ubReadFlash = 4  // bootloader can read flash
	ubChipErase = 16 // bootloader has chip erase
	ubNumMCU    = 2040

	mcuid328P  = 119
	mcuid328PB = 120
)

func Update(port, board string, cb func(format string, values ...interface{})) ([]byte, error) {
	baud := 115200
	if board == "Nano (old bootloader)" {
		baud = 57600
	}

	firmware, err := parseIntelHex(firmwareHex)
	if err != nil {
		return nil, fmt.Errorf("parse firmware: %w", err)
	}

	cb("%s", "Opening "+port+" ...")
	p, err := serial.Open(port, &serial.Mode{BaudRate: baud})
	if err != nil {
		return nil, err
	}
	defer p.Close()
	// Short per-read timeout so we can hammer GET_SYNC inside the brief
	// (~1s) bootloader window without overshooting it.
	p.SetReadTimeout(200 * time.Millisecond)

	// Arduino auto-reset: the reset cap triggers on the falling edge of DTR,
	// so assert high first to guarantee a clean high->low->high pulse
	// regardless of the line state the driver left on open.
	p.SetDTR(true)
	p.SetRTS(true)
	time.Sleep(50 * time.Millisecond)
	p.SetDTR(false)
	p.SetRTS(false)
	time.Sleep(250 * time.Millisecond)
	p.SetDTR(true)
	p.SetRTS(true)
	time.Sleep(50 * time.Millisecond)
	p.ResetInputBuffer()

	pr := &programmer{p: p}

	cb("%s", "Syncing with bootloader ...")
	if err := pr.sync(); err != nil {
		return nil, err
	}

	if pr.insync == stkInsync && pr.ok == stkOK {
		err = pr.flashOptiboot(firmware, cb)
	} else {
		err = pr.flashUrboot(firmware, cb)
	}
	if err != nil {
		return nil, err
	}

	cb("%s", "Done")
	return nil, nil
}

type programmer struct {
	p serial.Port
	// Protocol ack bytes learned during sync: 0x14/0x10 for STK500v1
	// (optiboot), anything else encodes urboot's MCU id and features.
	insync, ok byte
}

// sync hammers GET_SYNC until the bootloader answers with the same two ack
// bytes twice in a row. Bootloaders only listen for ~1s after reset, so we
// send fast with a short read timeout rather than waiting long on any single
// attempt. The first byte (0x30) is also what urboot's autobaud locks onto.
func (pr *programmer) sync() error {
	deadline := time.Now().Add(5 * time.Second)
	resp := make([]byte, 2)
	var last [2]byte
	seen := false
	for time.Now().Before(deadline) {
		pr.p.ResetInputBuffer() // drain: guards against line noise / app chatter
		if _, err := pr.p.Write([]byte{cmdGetSync, crcEOP}); err != nil {
			return err
		}
		if err := pr.readFull(resp); err != nil || resp[0] == resp[1] {
			seen = false
			continue // timeout/no data/garbage, try again
		}
		if seen && resp[0] == last[0] && resp[1] == last[1] {
			pr.insync, pr.ok = resp[0], resp[1]
			return nil
		}
		last[0], last[1] = resp[0], resp[1]
		seen = true
	}
	return fmt.Errorf("could not sync with bootloader (no response) - check the board/baud and that nothing else has the port open")
}

// flashOptiboot writes firmware through a classic STK500v1 bootloader.
func (pr *programmer) flashOptiboot(firmware []byte, cb func(string, ...interface{})) error {
	sig, err := pr.cmd([]byte{cmdReadSign}, 3)
	if err != nil {
		return fmt.Errorf("read signature: %w", err)
	}
	cb("Device signature: %02X %02X %02X", sig[0], sig[1], sig[2])
	// ATmega328P = 1E 95 0F, ATmega328PB = 1E 95 16. Same flash size and page size.
	if sig[0] != 0x1E || sig[1] != 0x95 || (sig[2] != 0x0F && sig[2] != 0x16) {
		return fmt.Errorf("unexpected device signature %02X%02X%02X, expected 1E950F (ATmega328P) or 1E9516 (ATmega328PB)", sig[0], sig[1], sig[2])
	}

	if _, err := pr.cmd([]byte{cmdEnterPM}, 0); err != nil {
		return fmt.Errorf("enter programming mode: %w", err)
	}

	cb("Writing %d bytes ...", len(firmware))
	for addr := 0; addr < len(firmware); addr += pageSize {
		end := addr + pageSize
		if end > len(firmware) {
			end = len(firmware)
		}
		if err := pr.writePage(addr, firmware[addr:end]); err != nil {
			return fmt.Errorf("write page at 0x%X: %w", addr, err)
		}
		cb("Wrote 0x%04X", addr)
	}

	if _, err := pr.cmd([]byte{cmdLeavePM}, 0); err != nil {
		return fmt.Errorf("leave programming mode: %w", err)
	}
	return nil
}

// decodeUrbootInfo extracts MCU id and feature bits from urboot's ack bytes.
func decodeUrbootInfo(insync, ok byte) (mcuid, features int) {
	o := int(ok)
	if o > int(insync) {
		o--
	}
	info := int(insync)*255 + o
	return info % ubNumMCU, info / ubNumMCU
}

// flashUrboot writes firmware through a urboot bootloader (MiniCore default)
// using urprotocol. Flash only; no metadata is written (avrdude -x nometadata).
func (pr *programmer) flashUrboot(firmware []byte, cb func(string, ...interface{})) error {
	mcuid, feat := decodeUrbootInfo(pr.insync, pr.ok)
	if mcuid != mcuid328P && mcuid != mcuid328PB {
		return fmt.Errorf("unexpected urboot MCU id %d, expected %d (ATmega328P) or %d (ATmega328PB)", mcuid, mcuid328P, mcuid328PB)
	}
	if feat&ubReadFlash == 0 {
		return fmt.Errorf("urboot bootloader cannot read flash, cannot locate it")
	}

	// Top 6 bytes of flash: numpages, vectnum, rjmp writepage (2), cap, version.
	const top = flashSize - 6
	info, err := pr.cmd([]byte{urReadPageFL, byte(top & 0xFF), byte(top >> 8), 6}, 6)
	if err != nil {
		return fmt.Errorf("read bootloader info: %w", err)
	}
	numpages, vectnum, urver := int(info[0]&0x7f), int(info[1]&0x7f), info[5]
	if urver < 0o72 || urver > 0o147 || numpages == 0 || numpages*pageSize > 2048 {
		return fmt.Errorf("unrecognised bootloader info % X", info)
	}
	blstart := flashSize - numpages*pageSize
	cb("urboot v%d.%d on MCU id %d, bootloader at 0x%04X, vector %d", urver>>3, urver&7, mcuid, blstart, vectnum)

	if len(firmware) > blstart {
		return fmt.Errorf("firmware (%d bytes) overlaps bootloader at 0x%04X", len(firmware), blstart)
	}
	fw := append([]byte(nil), firmware...)
	if vectnum > 0 {
		if err := patchVectors(fw, blstart, vectnum); err != nil {
			return err
		}
	}

	if feat&ubChipErase != 0 {
		cb("%s", "Erasing ...")
		pr.p.SetReadTimeout(10 * time.Second)
		_, err := pr.cmd([]byte{cmdChipErase}, 0)
		pr.p.SetReadTimeout(200 * time.Millisecond)
		if err != nil {
			return fmt.Errorf("chip erase: %w", err)
		}
	} else {
		// ponytail: emulate chip erase by writing 0xFF over the whole app area
		for len(fw) < blstart {
			fw = append(fw, 0xFF)
		}
	}

	cb("Writing %d bytes ...", len(fw))
	page := make([]byte, pageSize)
	for addr := 0; addr < len(fw); addr += pageSize {
		for i := range page {
			page[i] = 0xFF
		}
		copy(page, fw[addr:])
		payload := append([]byte{urProgPageFL, byte(addr), byte(addr >> 8), byte(pageSize)}, page...)
		if _, err := pr.cmd(payload, 0); err != nil {
			return fmt.Errorf("write page at 0x%X: %w", addr, err)
		}
		cb("Wrote 0x%04X", addr)
	}

	if _, err := pr.cmd([]byte{cmdLeavePM}, 0); err != nil {
		return fmt.Errorf("leave programming mode: %w", err)
	}
	return nil
}

// patchVectors turns fw into a vector-bootloader image: the reset vector
// jumps to the bootloader and vector slot vectnum jumps to the application's
// original start. Mirrors avrdude's urclock for 4-byte vectors (flash > 8K).
func patchVectors(fw []byte, blstart, vectnum int) error {
	const vecsz = 4
	if len(fw) < (vectnum+1)*vecsz {
		return fmt.Errorf("firmware too short to hold vector %d", vectnum)
	}
	op16 := uint16(fw[0]) | uint16(fw[1])<<8
	var app int
	switch {
	case op16&0xFE0E == 0x940C: // jmp k
		w := int(fw[2]) | int(fw[3])<<8
		app = (w | int(op16&1)<<16 | int(op16&0x1F0)<<13) << 1
	case op16&0xF000 == 0xC000: // rjmp k
		d := int(int16(op16<<4)>>3) + 2 // signed word offset -> bytes, relative to next insn
		d &= 8191
		if d >= 4096 {
			d -= 8192
		}
		if d < 0 {
			d += flashSize
		}
		app = d
	default:
		return fmt.Errorf("reset vector %04X is not a jmp/rjmp", op16)
	}
	if app == blstart {
		return nil // already points to the bootloader
	}
	if app < vectnum*vecsz || app >= len(fw) {
		return fmt.Errorf("reset vector jumps to 0x%04X, outside the application", app)
	}
	// Reset: rjmp backwards (wrapping around flash end) to bootloader + "ur" marker.
	r := 0xC000 | uint16((blstart-flashSize-2)/2)&0x0FFF
	fw[0], fw[1], fw[2], fw[3] = byte(r), byte(r>>8), 0x75, 0x72
	// Vector slot: jmp app.
	w := app >> 1
	j := uint16(0x940C) | uint16(w>>16&1) | uint16(w>>17&0x1F)<<4
	i := vectnum * vecsz
	fw[i], fw[i+1], fw[i+2], fw[i+3] = byte(j), byte(j>>8), byte(w), byte(w>>8)
	return nil
}

func (pr *programmer) writePage(addr int, data []byte) error {
	// STK500 addresses flash in words.
	word := addr / 2
	if _, err := pr.cmd([]byte{cmdLoadAddr, byte(word), byte(word >> 8)}, 0); err != nil {
		return err
	}
	payload := []byte{cmdProgPage, byte(len(data) >> 8), byte(len(data)), 'F'}
	payload = append(payload, data...)
	_, err := pr.cmd(payload, 0)
	return err
}

// cmd sends payload+CRC_EOP, then reads INSYNC, respLen data bytes, and OK.
func (pr *programmer) cmd(payload []byte, respLen int) ([]byte, error) {
	if _, err := pr.p.Write(append(payload, crcEOP)); err != nil {
		return nil, err
	}
	head := make([]byte, 1)
	if err := pr.readFull(head); err != nil {
		return nil, err
	}
	if head[0] != pr.insync {
		return nil, fmt.Errorf("expected INSYNC 0x%02X, got 0x%02X", pr.insync, head[0])
	}
	resp := make([]byte, respLen)
	if respLen > 0 {
		if err := pr.readFull(resp); err != nil {
			return nil, err
		}
	}
	tail := make([]byte, 1)
	if err := pr.readFull(tail); err != nil {
		return nil, err
	}
	if tail[0] != pr.ok {
		return nil, fmt.Errorf("expected OK 0x%02X, got 0x%02X", pr.ok, tail[0])
	}
	return resp, nil
}

func (pr *programmer) readFull(b []byte) error {
	for got := 0; got < len(b); {
		n, err := pr.p.Read(b[got:])
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrNoProgress // read timeout with no data
		}
		got += n
	}
	return nil
}

// parseIntelHex decodes Intel HEX into a flat byte slice, padding gaps with 0xFF.
func parseIntelHex(raw []byte) ([]byte, error) {
	var out []byte
	for _, line := range splitLines(raw) {
		if len(line) == 0 || line[0] != ':' {
			continue
		}
		b, err := hex.DecodeString(string(line[1:]))
		if err != nil {
			return nil, err
		}
		if len(b) < 5 {
			return nil, fmt.Errorf("short record")
		}
		count := int(b[0])
		if len(b) != count+5 {
			return nil, fmt.Errorf("bad record length")
		}
		var sum byte
		for _, x := range b {
			sum += x
		}
		if sum != 0 {
			return nil, fmt.Errorf("checksum error")
		}
		addr := int(b[1])<<8 | int(b[2])
		switch b[3] {
		case 0x00: // data
			end := addr + count
			for len(out) < end {
				out = append(out, 0xFF)
			}
			copy(out[addr:end], b[4:4+count])
		case 0x01: // EOF
			return out, nil
		default:
			return nil, fmt.Errorf("unsupported record type 0x%02X", b[3])
		}
	}
	return out, nil
}

func splitLines(raw []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == '\n' || raw[i] == '\r' {
			if i > start {
				lines = append(lines, raw[start:i])
			}
			start = i + 1
		}
	}
	return lines
}
