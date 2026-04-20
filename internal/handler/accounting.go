package handler

import (
    "log/slog"
    "net"
    "time"

    "layeh.com/radius"
    "layeh.com/radius/rfc2866"
    "layeh.com/radius/rfc2865"

    "radius-accounting-server/internal/config"
    "radius-accounting-server/internal/model"
    "radius-accounting-server/internal/store"
)

type AccountingHandler struct {
    store  store.Store
    cfg    *config.Config
    logger *slog.Logger
}

func InitAccountingHandler(storage store.Store, cfg *config.Config, logger *slog.Logger) *AccountingHandler {

    return &AccountingHandler {store: storage, 
                               cfg: cfg, 
			       logger: logger,}
}

//Handle the Reuqest 
func (acctHandler *AccountingHandler) ServeRADIUS(writer radius.ResponseWriter, request *radius.Request) {

    // Ensure we ALWAYS send a response to the NAS, even if processing fails locally
    defer func() {
      _ = writer.Write(request.Response(radius.CodeAccountingResponse))
    }()

    ctx := request.Context()

    //Build the Account Record from UDP Request
    accountRecord := acctHandler.buildRecord(request)

    if accountRecord == nil {
	// Even on parse failure, Respond with Accounting-Response
        acctHandler.logger.Error("Failed to build accounting record")
        return
    }

    //Save the created Account Record
    err := acctHandler.store.Save(ctx, accountRecord, acctHandler.cfg.SessionTTL)
    if err != nil {
        acctHandler.logger.Error("Failed to save accounting record, ", "Error:", err)
	return
    }
}

//Build Account Record
func (acctHandler *AccountingHandler) buildRecord(request *radius.Request) *model.AccountingRecord {

    // Get Client IP.
    clientIP, _, _ := net.SplitHostPort(request.RemoteAddr.String())

    //Get Port
    nasPort := rfc2865.NASPort_Get(request.Packet)

    return &model.AccountingRecord {
        Username:         rfc2865.UserName_GetString(request.Packet),
        NASIPAddress:     rfc2865.NASIPAddress_Get(request.Packet).String(),
        NASPort:          uint32(nasPort),
        AcctStatusType:   mapStatusType(rfc2866.AcctStatusType_Get(request.Packet)),
	AcctSessionID:    rfc2866.AcctSessionID_GetString(request.Packet),
        FramedIPAddress:  rfc2865.FramedIPAddress_Get(request.Packet).String(),
        CallingStationID: rfc2865.CallingStationID_GetString(request.Packet),
        CalledStationID:  rfc2865.CalledStationID_GetString(request.Packet),
        Timestamp:        time.Now().UTC(),
        ClientIP:         clientIP,
        PacketType:       request.Code.String(),
        AcctInputOctets:  uint64(rfc2866.AcctInputOctets_Get(request.Packet)),
        AcctOutputOctets: uint64(rfc2866.AcctOutputOctets_Get(request.Packet)),
        AcctSessionTime:  uint32(rfc2866.AcctSessionTime_Get(request.Packet)),
    }
}

//Converts numeric AcctStatusType from packet to human-readable model.AccountingStatusType string for storage
func mapStatusType(st rfc2866.AcctStatusType) model.AccountingStatusType {

    switch st {

     case rfc2866.AcctStatusType_Value_Start:         return model.StatusStart
     case rfc2866.AcctStatusType_Value_Stop:          return model.StatusStop
     case rfc2866.AcctStatusType_Value_InterimUpdate: return model.StatusInterim
     case rfc2866.AcctStatusType_Value_AccountingOn:  return model.StatusOn
     case rfc2866.AcctStatusType_Value_AccountingOff: return model.StatusOff
     default:                                          return model.StatusUnknown
   }
}

