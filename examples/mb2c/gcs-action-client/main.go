package main

import (
	"fmt"

	"github.com/p-akshay/diameter/v4/diam"
	"github.com/p-akshay/diameter/v4/diam/datatype"
	"github.com/p-akshay/diameter/v4/diam/dict"
)

func main() {
	peer := diam.MB2CPeer{
		OriginHost:       datatype.DiameterIdentity("bmsc.example.com"),
		OriginRealm:      datatype.DiameterIdentity("example.com"),
		DestinationHost:  datatype.DiameterIdentity("gcs.example.net"),
		DestinationRealm: datatype.DiameterIdentity("example.net"),
	}

	gar, err := diam.NewGCSActionRequest("session;12345", peer, dict.Default)
	if err != nil {
		panic(err)
	}

	diam.AddDefaultMB2CSupportedFeatures(gar)
	diam.AddTMGIAllocationRequest(gar, 1)

	fmt.Println(gar.PrettyDump())
}
