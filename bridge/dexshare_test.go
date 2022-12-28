package bridge

import (
	"ns-gobridge/common"
	"os"
	"reflect"
	"regexp"
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
	common.SetEnvWithAwsSSM("prod-bridge-secrets", "eu-west-1")
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

func TestGetSessionIdLength(t *testing.T) {
	common.SetEnvWithAwsSSM("prod-bridge-secrets", "eu-west-1")
	test_auth_url := "http://shareous1.dexcom.com/ShareWebServices/Services/General/AuthenticatePublisherAccount"
	test_login_url := "http://shareous1.dexcom.com/ShareWebServices/Services/General/LoginPublisherAccountById"
	type args struct {
		login_url string
		auth_url  string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "", args: args{auth_url: test_auth_url, login_url: test_login_url}, want: 36},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSessionId(tt.args.login_url, tt.args.auth_url)
			if len(got) != tt.want {
				t.Errorf("Length of GetSessionId() output = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSessionIdLFormat(t *testing.T) {
	common.SetEnvWithAwsSSM("prod-bridge-secrets", "eu-west-1")
	test_auth_url := "http://shareous1.dexcom.com/ShareWebServices/Services/General/AuthenticatePublisherAccount"
	test_login_url := "http://shareous1.dexcom.com/ShareWebServices/Services/General/LoginPublisherAccountById"
	type args struct {
		login_url string
		auth_url  string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "", args: args{auth_url: test_auth_url, login_url: test_login_url}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSessionId(tt.args.login_url, tt.args.auth_url)
			strFormat := "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
			r := regexp.MustCompile(strFormat)
			if r.MatchString(got) != tt.want {
				t.Errorf("Format of GetSessionId() output = %v, want %v", got, strFormat)
			}
		})
	}
}

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
