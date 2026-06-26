package media

import (
	"testing"
)

func TestStringArray_Value(t *testing.T) {
	tests := []struct {
		name string
		in   StringArray
		want string
	}{
		{name: "empty", in: StringArray{}, want: "{}"},
		{name: "single", in: StringArray{"4k"}, want: `{"4k"}`},
		{name: "multi", in: StringArray{"1080p", "BluRay", "X264"}, want: `{"1080p","BluRay","X264"}`},
		{name: "quote", in: StringArray{`say "hi"`}, want: `{"say \"hi\""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Value()
			if err != nil {
				t.Fatal(err)
			}
			s, ok := got.(string)
			if !ok {
				t.Fatalf("Value() = %T, want string", got)
			}
			if s != tt.want {
				t.Fatalf("Value() = %q, want %q", s, tt.want)
			}
		})
	}
}

func TestStringArray_Scan(t *testing.T) {
	var arr StringArray
	if err := arr.Scan([]byte(`{"1080p","BluRay"}`)); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 || arr[0] != "1080p" || arr[1] != "BluRay" {
		t.Fatalf("Scan() = %#v", []string(arr))
	}
}
