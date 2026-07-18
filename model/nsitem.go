package model

import (
	"time"

	"github.com/uptrace/bun"
)

type DexBgReading struct {
	WT    string `json:"WT"`
	ST    string `json:"ST"`
	DT    string `json:"DT"`
	Value int    `json:"Value"`
	Trend string `json:"Trend"`
}

type NsBgEntry struct {
	Sgv        int    `json:"sgv"`
	Date       int64  `json:"date"`
	DateString string `json:"dateString"`
	Trend      int    `json:"trend"`
	Direction  string `json:"direction"`
	Device     string `json:"device"`
	Type       string `json:"type"`
	UtcOffset  int    `json:"utcOffset"`
	Hash       string `json:"hash"`
}

type Nightscoutdb struct {
	bun.BaseModel `bun:"table:nightscoutdb,alias:ns"`

	Id          int64     `bun:"id,pk,autoincrement"`
	Sgv         int       `bun:"sgv"`
	Ns_time     int64     `bun:"ns_time,type:bigint"`
	Ns_datetime time.Time `bun:"ns_datetime,type:timestampz"`
	Trend       int       `bun:"trend"`
	Utcoffset   int       `bun:"utcoffset"`
	Systime     time.Time `bun:"systime,type:timestampz"`
}
