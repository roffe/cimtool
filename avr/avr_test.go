package avr

import "testing"

func TestParseIntelHex(t *testing.T) {
	in := []byte(":100000000C9462000C948A000C948A000C948A0070\n:00000001FF\n")
	out, err := parseIntelHex(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 16 || out[0] != 0x0C || out[1] != 0x94 || out[2] != 0x62 {
		t.Fatalf("bad decode: %X", out)
	}

	// embedded firmware must decode and start with a reset vector jump (0x0C 0x94)
	fw, err := parseIntelHex(firmwareHex)
	if err != nil {
		t.Fatal(err)
	}
	if len(fw) == 0 || fw[0] != 0x0C || fw[1] != 0x94 {
		t.Fatalf("firmware decode looks wrong: len=%d head=%X", len(fw), fw[:2])
	}

	// flipped checksum byte must fail
	if _, err := parseIntelHex([]byte(":100000000C9462000C948A000C948A000C948A0071\n")); err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestDecodeUrbootInfo(t *testing.T) {
	// Encode the way urboot does: info = features*2040 + mcuid, split into
	// insync/ok with ok bumped past insync so the two ack bytes differ.
	for _, tc := range []struct{ mcuid, feat int }{{mcuid328PB, ubReadFlash | ubChipErase}, {mcuid328P, 0}, {2039, 31}} {
		info := tc.feat*ubNumMCU + tc.mcuid
		insync, ok := byte(info/255), byte(info%255)
		if ok >= insync {
			ok++
		}
		m, f := decodeUrbootInfo(insync, ok)
		if m != tc.mcuid || f != tc.feat {
			t.Fatalf("decode(%02X %02X) = %d,%d want %d,%d", insync, ok, m, f, tc.mcuid, tc.feat)
		}
	}
}

func TestPatchVectors(t *testing.T) {
	// MiniCore 3.1.3 urboot 8.0 for 328PB: 384 bytes at 0x7E80, vector 25.
	const blstart, vectnum = 0x7E80, 25
	fw := make([]byte, 512)
	copy(fw, []byte{0x0C, 0x94, 0x80, 0x00}) // jmp 0x0100
	if err := patchVectors(fw, blstart, vectnum); err != nil {
		t.Fatal(err)
	}
	// rjmp -193 words wraps to 0x7E80 (urboot formula), followed by "ur".
	if want := []byte{0x3F, 0xCF, 0x75, 0x72}; string(fw[:4]) != string(want) {
		t.Fatalf("reset = % X want % X", fw[:4], want)
	}
	if want := []byte{0x0C, 0x94, 0x80, 0x00}; string(fw[100:104]) != string(want) {
		t.Fatalf("vector 25 = % X want % X", fw[100:104], want)
	}
	// Already pointing at the bootloader: untouched.
	before := append([]byte(nil), fw...)
	if err := patchVectors(fw, blstart, vectnum); err != nil || string(fw) != string(before) {
		t.Fatalf("second patch changed image or failed: %v", err)
	}
	// rjmp reset (small app) decodes to the same destination.
	fw2 := make([]byte, 512)
	copy(fw2, []byte{0x7F, 0xC0}) // rjmp .+254 -> 0x0100
	if err := patchVectors(fw2, blstart, vectnum); err != nil {
		t.Fatal(err)
	}
	if string(fw2[100:104]) != string(fw[100:104]) {
		t.Fatalf("rjmp reset: vector 25 = % X", fw2[100:104])
	}
	// Not a jump: refuse.
	if err := patchVectors(make([]byte, 512), blstart, vectnum); err == nil {
		t.Fatal("expected error for 0x0000 reset vector")
	}
}
