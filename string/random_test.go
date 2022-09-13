package strutil

import (
	"reflect"
	"testing"
)

func TestRandBytes(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "1-测试10个byte",
			args:    args{10},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "2-测试10个byte",
			args:    args{10},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RandBytes(tt.args.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("RandBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RandBytes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandString(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "1-测试10个字符",
			args:    args{10},
			want:    16,
			wantErr: false,
		},
		{
			name:    "2-测试16个字符",
			args:    args{16},
			want:    24,
			wantErr: false,
		},
		{
			name:    "3-测试32个字符",
			args:    args{32},
			want:    44,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RandUrlEncodingString(tt.args.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("RandString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("RandString() got = %v, want %v, len(got) = %v", got, tt.want, len(got))
			}
		})
	}
	t.Logf("byte: %v", []byte("0123456789"))
}

func TestRandStringNumber(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "1-test",
			args: args{20},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RandStringNumber(tt.args.n); got != tt.want {
				t.Errorf("RandStringNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}
