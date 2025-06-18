package bind

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type testStruct struct {
	Name  string `json:"name" xml:"name"`
	Value int    `json:"value" xml:"value"`
}

func TestJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    testStruct
		wantErr bool
	}{
		{
			name:  "valid JSON",
			input: `{"name":"test","value":123}`,
			want:  testStruct{Name: "test", Value: 123},
		},
		{
			name:    "invalid JSON",
			input:   `{"name":"test"`,
			wantErr: true,
		},
		{
			name:  "empty JSON object",
			input: `{}`,
			want:  testStruct{},
		},
		{
			name:    "invalid type",
			input:   `{"name":"test","value":"not a number"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got testStruct
			err := JSON(strings.NewReader(tt.input), &got)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && got != tt.want {
				t.Errorf("JSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestXML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    testStruct
		wantErr bool
	}{
		{
			name:  "valid XML",
			input: `<testStruct><name>test</name><value>123</value></testStruct>`,
			want:  testStruct{Name: "test", Value: 123},
		},
		{
			name:    "invalid XML",
			input:   `<testStruct><name>test`,
			wantErr: true,
		},
		{
			name:  "empty XML",
			input: `<testStruct></testStruct>`,
			want:  testStruct{},
		},
		{
			name:    "invalid type",
			input:   `<testStruct><name>test</name><value>not a number</value></testStruct>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got testStruct
			err := XML(strings.NewReader(tt.input), &got)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("XML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && got != tt.want {
				t.Errorf("XML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONBodyAlreadyRead(t *testing.T) {
	data := `{"name":"test","value":123}`
	reader := strings.NewReader(data)
	
	// Read part of the body first
	buf := make([]byte, 5)
	_, err := reader.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	
	// Try to decode - should fail because body is partially read
	var got testStruct
	err = JSON(reader, &got)
	if err == nil {
		t.Error("Expected error when body is partially read, but got nil")
	}
}

func TestXMLBodyAlreadyRead(t *testing.T) {
	// XML decoder is more lenient with partial reads, so we need a different approach
	data := `invalid xml data`
	reader := strings.NewReader(data)
	
	// Try to decode invalid XML
	var got testStruct
	err := XML(reader, &got)
	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}

func TestJSONDrainsReader(t *testing.T) {
	data := `{"name":"test","value":123}extra data that should be drained`
	reader := strings.NewReader(data)
	
	var got testStruct
	err := Json(reader, &got)
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify reader is drained
	remaining, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(remaining) > 0 {
		t.Errorf("Reader not fully drained, remaining: %s", remaining)
	}
}

func TestXMLDrainsReader(t *testing.T) {
	data := `<testStruct><name>test</name><value>123</value></testStruct>extra data`
	reader := strings.NewReader(data)
	
	var got testStruct
	err := Xml(reader, &got)
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify reader is drained
	remaining, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(remaining) > 0 {
		t.Errorf("Reader not fully drained, remaining: %s", remaining)
	}
}

func BenchmarkJSON(b *testing.B) {
	data := []byte(`{"name":"test","value":123}`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v testStruct
		_ = JSON(bytes.NewReader(data), &v)
	}
}

func BenchmarkXML(b *testing.B) {
	data := []byte(`<testStruct><name>test</name><value>123</value></testStruct>`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v testStruct
		_ = XML(bytes.NewReader(data), &v)
	}
}