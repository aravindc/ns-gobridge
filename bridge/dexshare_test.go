package bridge

import (
	"os"
	"reflect"
	"testing"
)

func TestGetDexServer(t *testing.T) {
	os.Setenv("BRIDGE_SERVER", "US")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Dexcom Server value for US",
			want: "share2.dexcom.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDexServer(); got != tt.want {
				t.Errorf("GetDexServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getPayload(t *testing.T) {
	os.Setenv("BRIDGE_USER", "user001")
	os.Setenv("BRIDGE_PASS", "pass001")
	os.Setenv("APPLICATION_ID", "app001")
	type args struct {
		input_type   string
		input_string string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Payload Test1",
			args: args{
				input_type:   "accountNum",
				input_string: os.Getenv("BRIDGE_USER"),
			},
			want: []byte(`{"password": "pass001", "applicationId": "app001", "accountNum": "user001"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPayload(tt.args.input_type, tt.args.input_string); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPayload() = %v, want %v", string(got), string(tt.want))
			}
		})
	}

	os.Unsetenv("BRIDGE_USER")
	os.Unsetenv("BRIDGE_PASS")
	os.Unsetenv("APPLICATION_ID")
}
