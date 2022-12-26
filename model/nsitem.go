package model

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
