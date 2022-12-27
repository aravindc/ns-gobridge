package bridge

import (
	"ns-gobridge/common"
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

func TestGetAccountId(t *testing.T) {
	common.SetEnvWithAwsSecret()
	test_auth_url := "http://shareous1.dexcom.com/ShareWebServices/Services/General/AuthenticatePublisherAccount"
	type args struct {
		auth_url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "", args: args{auth_url: test_auth_url}, want: "2b7646cf-73ba-4c19-8463-f4fbda8c2af4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAccountId(tt.args.auth_url); got != tt.want {
				t.Errorf("GetAccountId() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func TestGetSessionId(t *testing.T) {
// 	type args struct {
// 		login_url string
// 		auth_url  string
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want string
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := GetSessionId(tt.args.login_url, tt.args.auth_url); got != tt.want {
// 				t.Errorf("GetSessionId() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestGetLatestBG(t *testing.T) {
// 	type args struct {
// 		latestbg_url string
// 		session_id   string
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want []model.NsBgEntry
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := GetLatestBG(tt.args.latestbg_url, tt.args.session_id); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("GetLatestBG() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
