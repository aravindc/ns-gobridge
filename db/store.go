package db

import (
	"encoding/json"
	"ns-gobridge/model"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/supabase/postgrest-go"
)

func DbClient(supabase_connection_string string) *postgrest.Client {

	client := postgrest.NewClient(supabase_connection_string, "public", nil)
	if client.ClientError != nil {
		log.Fatal(client.ClientError)
	}
	client.TokenAuth(os.Getenv("SUPABASE_KEY"))
	return client
}

func SelectEntries(db_client *postgrest.Client) []model.NsBgEntry {
	res, _, err := db_client.From("entries").Select("sgv,date,dateString,trend,direction,device,type,hash", "", false).Execute()
	if err != nil {
		log.Fatal(err)
	}
	var data []model.NsBgEntry
	err = json.Unmarshal(res, &data)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func EntriesExist(db_client *postgrest.Client, nsBgEntry model.NsBgEntry) bool {
	res, _, err := db_client.From("entries").Select("sgv,date,dateString,trend,direction,device,type,hash", "", false).Eq("hash", nsBgEntry.Hash).Execute()
	if err != nil {
		log.Fatal(err)
	}
	var data []model.NsBgEntry
	err = json.Unmarshal(res, &data)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(data)
	return len(data) > 0
}

func InsertEntries(db_client *postgrest.Client, nsItem model.NsBgEntry) {
	_, _, err := db_client.From("entries").Insert(nsItem, false, "", "minimal", "").Execute()
	if err != nil {
		log.Fatal(err)
	}
}
