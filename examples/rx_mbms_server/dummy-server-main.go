package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/p-akshay/diameter/v4/diam"
	"github.com/p-akshay/diameter/v4/diam/avp"
	"github.com/p-akshay/diameter/v4/diam/datatype"
	"github.com/p-akshay/diameter/v4/diam/sm"
)

const defaultTMGI = datatype.OctetString("000001234567")

func main() {
	addr := flag.String("addr", ":3868", "Diameter listen address")
	host := flag.String("diam_host", "dummy-diameter.local", "Origin-Host used by this dummy server")
	realm := flag.String("diam_realm", "local", "Origin-Realm used by this dummy server")
	verbose := flag.Bool("v", true, "print full incoming and outgoing Diameter messages")
	flag.Parse()

	settings := &sm.Settings{
		OriginHost: datatype.DiameterIdentity(*host),
		OriginRealm: datatype.DiameterIdentity(*realm),
		VendorID: diam.TGPP_VENDOR_ID,
		ProductName: "dummy-rx-mb2c-server",
		FirmwareRevision: 1,
	}

	mux := sm.New(settings)
	mux.HandleFunc(diam.AAR, handleRxAAR(settings, *verbose))
	mux.HandleFunc(diam.GAR, handleMB2CGAR(settings, *verbose))
	mux.HandleFunc("ALL", handleAll)

	go printErrors(mux.ErrorReports())

	log.Printf("dummy Diameter server listening on %s origin=%s realm=%s", *addr, *host, *realm)
	log.Printf("supported handlers: Rx AAR/AAA app=%d, MB2-C GAR/GAA app=%d", diam.RX_APP_ID, diam.MB2C_APP_ID)
	if err := diam.ListenAndServe(*addr, mux, nil); err != nil { log.Fatal(err) }
}

func printErrors(ec <-chan *diam.ErrorReport) { for err := range ec { log.Printf("diameter error: %v", err) } }

func handleRxAAR(settings *sm.Settings, verbose bool) diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		log.Printf("RX: received AAR from %s session=%s app=%d", c.RemoteAddr(), avpString(m, avp.SessionID), m.Header.ApplicationID)
		logInterestingAVPs("RX", m, []interface{}{avp.AFAppId, avp.MediaCompDescp, avp.RxRequestType, avp.SubscriptionID})
		if verbose { log.Printf("RX: incoming AAR\n%s", m) }
		a := m.Answer(diam.Success)
		copyAVP(a, m, avp.SessionID, 0)
		addServerOrigin(a, settings)
		copyAVP(a, m, avp.AuthApplicationID, 0)
		copyAVP(a, m, avp.AuthSessionState, 0)
		writeAnswer("RX", c, a, verbose)
	}
}

func handleMB2CGAR(settings *sm.Settings, verbose bool) diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		log.Printf("MB2-C: received GAR from %s session=%s app=%d", c.RemoteAddr(), avpString(m, avp.SessionID), m.Header.ApplicationID)
		logInterestingAVPs("MB2-C", m, []interface{}{avp.SupportedFeatures, avp.TMGIAllocationRequest, avp.TMGIDeallocationRequest, avp.TMGINumber, avp.TMGI})
		if verbose { log.Printf("MB2-C: incoming GAR\n%s", m) }
		a := m.Answer(diam.Success)
		copyAVP(a, m, avp.SessionID, 0)
		copyAVP(a, m, avp.AuthApplicationID, 0)
		copyAVP(a, m, avp.AuthSessionState, 0)
		addServerOrigin(a, settings)
		addMB2CActionResponse(a, m)
		writeAnswer("MB2-C", c, a, verbose)
	}
}

func addMB2CActionResponse(a, req *diam.Message) {
	if _, err := req.FindAVP(avp.TMGIAllocationRequest, diam.TGPP_VENDOR_ID); err == nil {
		a.AddAVP(diam.NewAVP(avp.TMGIAllocationResponse, avp.Mbit|avp.Vbit, diam.TGPP_VENDOR_ID, &diam.GroupedAVP{AVP: []*diam.AVP{
			diam.NewAVP(avp.TMGI, avp.Vbit, diam.TGPP_VENDOR_ID, firstTMGI(req)),
			diam.NewAVP(avp.TMGIAllocationResult, avp.Mbit|avp.Vbit, diam.TGPP_VENDOR_ID, datatype.Unsigned32(0)),
		}}))
		return
	}
	if _, err := req.FindAVP(avp.TMGIDeallocationRequest, diam.TGPP_VENDOR_ID); err == nil {
		a.AddAVP(diam.NewAVP(avp.TMGIDeallocationResponse, avp.Mbit|avp.Vbit, diam.TGPP_VENDOR_ID, &diam.GroupedAVP{AVP: []*diam.AVP{
			diam.NewAVP(avp.TMGIDeallocationResult, avp.Mbit|avp.Vbit, diam.TGPP_VENDOR_ID, datatype.Unsigned32(0)),
		}}))
	}
}

func firstTMGI(m *diam.Message) datatype.OctetString {
	if a, err := m.FindAVP(avp.TMGI, diam.TGPP_VENDOR_ID); err == nil { if v, ok := a.Data.(datatype.OctetString); ok && v != "" { return v } }
	return defaultTMGI
}

func addServerOrigin(m *diam.Message, settings *sm.Settings) { m.NewAVP(avp.OriginHost, avp.Mbit, 0, settings.OriginHost); m.NewAVP(avp.OriginRealm, avp.Mbit, 0, settings.OriginRealm) }
func copyAVP(dst, src *diam.Message, code interface{}, vendorID uint32) { if a, err := src.FindAVP(code, vendorID); err == nil && a != nil { dst.AddAVP(a) } }
func avpString(m *diam.Message, code interface{}) string { if a, err := m.FindAVP(code, 0); err == nil && a != nil && a.Data != nil { return a.Data.String() }; return "" }

func logInterestingAVPs(prefix string, m *diam.Message, codes []interface{}) {
	var parts []string
	for _, code := range codes {
		avps, err := m.FindAVPs(code, diam.TGPP_VENDOR_ID)
		if err != nil || len(avps) == 0 { avps, err = m.FindAVPs(code, 0) }
		if err == nil && len(avps) > 0 { parts = append(parts, fmt.Sprint(code)+"="+avps[0].Data.String()) }
	}
	if len(parts) > 0 { log.Printf("%s: key AVPs: %s", prefix, strings.Join(parts, ", ")) }
}

func writeAnswer(prefix string, c diam.Conn, a *diam.Message, verbose bool) { if verbose { log.Printf("%s: outgoing answer\n%s", prefix, a) }; if _, err := a.WriteTo(c); err != nil { log.Printf("%s: failed to write answer to %s: %v", prefix, c.RemoteAddr(), err) } }
func handleAll(c diam.Conn, m *diam.Message) { log.Printf("received unhandled Diameter message from %s cmd=%d app=%d flags=0x%x\n%s", c.RemoteAddr(), m.Header.CommandCode, m.Header.ApplicationID, m.Header.CommandFlags, m) }
