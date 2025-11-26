package http3

import (
	"bytes"
	"testing"
)

func TestDataFrame(t *testing.T) {
	data := []byte("Hello, HTTP/3!")
	frame := &DataFrame{Data: data}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	r := bytes.NewReader(buf)
	parsed, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	df, ok := parsed.(*DataFrame)
	if !ok {
		t.Fatalf("ParseFrame() returned wrong type")
	}

	if !bytes.Equal(df.Data, data) {
		t.Errorf("Data = %v, want %v", df.Data, data)
	}
}

func TestHeadersFrame(t *testing.T) {
	headerBlock := []byte{0x00, 0x01, 0x02, 0x03}
	frame := &HeadersFrame{HeaderBlock: headerBlock}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	r := bytes.NewReader(buf)
	parsed, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	hf, ok := parsed.(*HeadersFrame)
	if !ok {
		t.Fatalf("ParseFrame() returned wrong type")
	}

	if !bytes.Equal(hf.HeaderBlock, headerBlock) {
		t.Errorf("HeaderBlock = %v, want %v", hf.HeaderBlock, headerBlock)
	}
}

func TestSettingsFrame(t *testing.T) {
	settings := []Setting{
		{ID: SettingQPackMaxTableCapacity, Value: 4096},
		{ID: SettingMaxFieldSectionSize, Value: 16384},
		{ID: SettingQPackBlockedStreams, Value: 100},
	}

	frame := &SettingsFrame{Settings: settings}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	r := bytes.NewReader(buf)
	parsed, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	sf, ok := parsed.(*SettingsFrame)
	if !ok {
		t.Fatalf("ParseFrame() returned wrong type")
	}

	if len(sf.Settings) != len(settings) {
		t.Errorf("Settings count = %d, want %d", len(sf.Settings), len(settings))
	}

	for i, s := range settings {
		if sf.Settings[i].ID != s.ID || sf.Settings[i].Value != s.Value {
			t.Errorf("Setting[%d] = {%d, %d}, want {%d, %d}",
				i, sf.Settings[i].ID, sf.Settings[i].Value, s.ID, s.Value)
		}
	}

	// Test GetSetting
	value, ok := sf.GetSetting(SettingQPackMaxTableCapacity)
	if !ok {
		t.Error("GetSetting() not found")
	}
	if value != 4096 {
		t.Errorf("GetSetting() = %d, want 4096", value)
	}
}

func TestGoAwayFrame(t *testing.T) {
	frame := &GoAwayFrame{StreamID: 12345}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	r := bytes.NewReader(buf)
	parsed, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	gf, ok := parsed.(*GoAwayFrame)
	if !ok {
		t.Fatalf("ParseFrame() returned wrong type")
	}

	if gf.StreamID != frame.StreamID {
		t.Errorf("StreamID = %d, want %d", gf.StreamID, frame.StreamID)
	}
}

func TestCancelPushFrame(t *testing.T) {
	frame := &CancelPushFrame{PushID: 42}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	r := bytes.NewReader(buf)
	parsed, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	cf, ok := parsed.(*CancelPushFrame)
	if !ok {
		t.Fatalf("ParseFrame() returned wrong type")
	}

	if cf.PushID != frame.PushID {
		t.Errorf("PushID = %d, want %d", cf.PushID, frame.PushID)
	}
}

func TestMaxPushIDFrame(t *testing.T) {
	frame := &MaxPushIDFrame{PushID: 1000}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	r := bytes.NewReader(buf)
	parsed, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	mf, ok := parsed.(*MaxPushIDFrame)
	if !ok {
		t.Fatalf("ParseFrame() returned wrong type")
	}

	if mf.PushID != frame.PushID {
		t.Errorf("PushID = %d, want %d", mf.PushID, frame.PushID)
	}
}

func TestPushPromiseFrame(t *testing.T) {
	frame := &PushPromiseFrame{
		PushID:      42,
		HeaderBlock: []byte{0x01, 0x02, 0x03},
	}

	// Encode
	buf, err := frame.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo() error = %v", err)
	}

	// Decode
	r := bytes.NewReader(buf)
	parsed, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}

	pf, ok := parsed.(*PushPromiseFrame)
	if !ok {
		t.Fatalf("ParseFrame() returned wrong type")
	}

	if pf.PushID != frame.PushID {
		t.Errorf("PushID = %d, want %d", pf.PushID, frame.PushID)
	}

	if !bytes.Equal(pf.HeaderBlock, frame.HeaderBlock) {
		t.Errorf("HeaderBlock = %v, want %v", pf.HeaderBlock, frame.HeaderBlock)
	}
}

func TestVarIntEncoding(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{"1-byte", 42},
		{"2-byte", 1000},
		{"4-byte", 100000},
		{"8-byte", 10000000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			buf := appendVarInt(nil, tt.value)

			// Decode
			r := bytes.NewReader(buf)
			decoded, err := readVarInt(r)
			if err != nil {
				t.Fatalf("readVarInt() error = %v", err)
			}

			if decoded != tt.value {
				t.Errorf("readVarInt() = %d, want %d", decoded, tt.value)
			}

			// Check length
			expectedLen := varIntLen(tt.value)
			if uint64(len(buf)) != expectedLen {
				t.Errorf("encoded length = %d, want %d", len(buf), expectedLen)
			}
		})
	}
}

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	if len(settings.Settings) == 0 {
		t.Error("DefaultSettings() returned empty settings")
	}

	// Check for expected settings
	expectedSettings := map[uint64]uint64{
		SettingQPackMaxTableCapacity: 4096,
		SettingMaxFieldSectionSize:   16384,
		SettingQPackBlockedStreams:   100,
		SettingH3Datagram:            1,
	}

	for id, expectedValue := range expectedSettings {
		value, ok := settings.GetSetting(id)
		if !ok {
			t.Errorf("DefaultSettings() missing setting %d", id)
			continue
		}
		if value != expectedValue {
			t.Errorf("Setting %d = %d, want %d", id, value, expectedValue)
		}
	}
}

func BenchmarkDataFrameEncode(b *testing.B) {
	data := make([]byte, 1024)
	frame := &DataFrame{Data: data}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.AppendTo(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataFrameDecode(b *testing.B) {
	data := make([]byte, 1024)
	frame := &DataFrame{Data: data}
	buf, _ := frame.AppendTo(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(buf)
		_, err := ParseFrame(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSettingsFrameEncode(b *testing.B) {
	frame := DefaultSettings()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.AppendTo(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVarIntEncode(b *testing.B) {
	values := []uint64{42, 1000, 100000, 10000000000}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, v := range values {
			_ = appendVarInt(nil, v)
		}
	}
}
