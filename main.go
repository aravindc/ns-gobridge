package main

import (
	"ns-gobridge/bridge"
	"ns-gobridge/common"
	"ns-gobridge/db"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
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

	// Get base path where executable is stored
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path: ", err)
	}

	// Load .env as environment variables from base path
	_, envExists := os.LookupEnv("NS_ENV")
	if !envExists {
		log.Fatal("Environment variable: NS_ENV indicating the environment should be set")
	}
	env := os.Getenv("NS_ENV")
	if env == "" || env == "development" {
		env = ".env.development"
	} else if env == "production" {
		env = ".env"
	} else if env == "test" {
		env = ".env.test"
	} else {
		log.Fatal("Environment should be development / test / production")
	}

	log.Info("About to load env file: ", env)

	envPath := filepath.Join(filepath.Dir(exePath), env)
	_err := godotenv.Load(envPath)
	if _err != nil {
		log.Fatal("Unable to load .env file ", err)
	}

	// If AWS_REGION not available, default to eu-west-1
	_, awsRegionExists := os.LookupEnv("AWS_REGION")
	if !awsRegionExists {
		os.Setenv("AWS_REGION", "eu-west-1")
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

	supabase_client := db.DbClient(os.Getenv("SUPABASE_URI"))

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
