package main

import (
	"ns-gobridge/bridge"
	"ns-gobridge/common"
	"ns-gobridge/db"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

func init() {
	// Setup Log
	log.SetOutput(os.Stdout)
	log.SetFormatter(
		&easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05",
			LogFormat:       "[%lvl%]: %time% - %msg%\n",
		},
	)
	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	_, awsAccessKeyExists := os.LookupEnv("AWS_ACCESS_KEY")
	if !awsAccessKeyExists {
		log.Fatal("Environment variables: AWS_ACCESS_KEY shoud be set")
	}

	_, awsSecretKeyExists := os.LookupEnv("AWS_SECRET_KEY")
	if !awsSecretKeyExists {
		log.Fatal("Environment variables: AWS_SECRET_KEY shoud be set")
	}

	// If AWS_REGION not available, default to eu-west-1
	_, awsRegionExists := os.LookupEnv("AWS_REGION")
	if !awsRegionExists {
		os.Setenv("AWS_REGION", "eu-west-1")
	}

	_, recordCountExists := os.LookupEnv("RECORD_COUNT")
	if !recordCountExists {
		os.Setenv("RECORD_COUNT", "3")
	}

	// Set Env with AWS SSM
	common.SetEnvWithAwsSSM(os.Getenv("AWS_PARAMETER_NAME"), os.Getenv("AWS_REGION"))

	// Check if BRIDGE_SERVER variable value is set
	_, bridgeServerExists := os.LookupEnv("BRIDGE_SERVER")
	if !bridgeServerExists {
		log.Fatal("Environment variables: BRIDGE_SERVER shoud be set")
	}
	// Check if BRIDGE_USER variable value is set
	_, bridgeUserExists := os.LookupEnv("BRIDGE_USER")
	if !bridgeUserExists {
		log.Fatal("Environment variables: BRIDGE_USER shoud be set")
	}
	// Check if BRIDGE_PASS variable value is set
	_, bridgePassExists := os.LookupEnv("BRIDGE_PASS")
	if !bridgePassExists {
		log.Fatal("Environment variables: BRIDGE_PASS shoud be set")
	}
}

// Retrieve BG data every 2 minutes
func getBGData() {
	base_url := "https://" + bridge.GetDexServer()
	auth_url := base_url + "/ShareWebServices/Services/General/AuthenticatePublisherAccount"
	login_url := base_url + "/ShareWebServices/Services/General/LoginPublisherAccountById"
	latestbg_url := base_url + "/ShareWebServices/Services/Publisher/ReadPublisherLatestGlucoseValues"

	// Session is initialized only once and may become invalid after its expiry
	// TODO: Write a function to check the validity of session_id and renew if required
	session_id := bridge.GetSessionId(login_url, auth_url)
	latest_bg := bridge.GetLatestBG(latestbg_url, session_id)

	supabase_client := db.DbClient(os.Getenv("SUPABASE_SERVER"))

	// Initial Connection
	for _, val := range latest_bg {
		// Check if record exists based on hash value
		if !db.EntriesExist(supabase_client, val) {
			// Insert record into db if it does exist
			db.InsertEntries(supabase_client, val)
		}
	}

	// Run this ticker every 2 minutes
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		latest_bg := bridge.GetLatestBG(latestbg_url, session_id)
		for _, val := range latest_bg {
			if !db.EntriesExist(supabase_client, val) {
				db.InsertEntries(supabase_client, val)
			}
		}
	}
}

func main() {
	go getBGData()
	// Insert other function calls here
	select {}
}
