package model

import (
  "encoding/json"
  "fmt"
  "time"
)

type AccountingStatusType string

const (
    StatusStart    AccountingStatusType = "Accounting-Start"
    StatusStop     AccountingStatusType = "Accounting-Stop"
    StatusInterim  AccountingStatusType = "Accounting-Interim-Update"
    StatusOn       AccountingStatusType = "Accounting-On"
    StatusOff      AccountingStatusType = "Accounting-Off"
    StatusUnknown  AccountingStatusType = "Unknown"
)

type AccountingRecord struct {
    Username          string               `json:"username"`
    NASIPAddress      string               `json:"nas_ip_address"`
    NASPort           uint32               `json:"nas_port"`
    AcctStatusType    AccountingStatusType `json:"acct_status_type"`
    AcctSessionID     string               `json:"acct_session_id"`
    FramedIPAddress   string               `json:"framed_ip_address"`
    CallingStationID  string               `json:"calling_station_id"`
    CalledStationID   string               `json:"called_station_id"`
    Timestamp         time.Time            `json:"timestamp"`
    ClientIP           string               `json:"client_ip"`
    PacketType        string               `json:"packet_type"`
    AcctInputOctets   uint64               `json:"acct_input_octets"`
    AcctOutputOctets  uint64               `json:"acct_output_octets"`
    AcctSessionTime   uint32               `json:"acct_session_time"`
}

func (record *AccountingRecord) Key() string {

    user := record.Username
    if user == "" {
        user = "unknown"
    }

    sessionID := record.AcctSessionID

    if sessionID == "" {
        sessionID = "no-session"
    }

    timeStamp := record.Timestamp.UTC().Format("20060102T150405.000000")

    return fmt.Sprintf("radius:acct:%s:%s:%s", user, sessionID, timeStamp)
}

func (record *AccountingRecord) Marshal() ([]byte, error) {
    return json.Marshal(record)
}

func (record *AccountingRecord) Unmarshal (data []byte) error {
    return json.Unmarshal(data, record)
}


