// Copyright 2013-2018 go-diameter authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package diam

import (
	"fmt"

	"github.com/p-akshay/diameter/v4/diam/avp"
	"github.com/p-akshay/diameter/v4/diam/datatype"
	"github.com/p-akshay/diameter/v4/diam/dict"
)

const (
	// TGPP_VENDOR_ID is the 3GPP vendor id used by MB2-C AVPs.
	TGPP_VENDOR_ID = 10415

	// AuthSessionStateNoStateMaintained is used by MB2-C GCS-Action messages.
	AuthSessionStateNoStateMaintained datatype.Enumerated = 1
)

// MB2CPeer identifies the local and remote Diameter peer metadata needed to
// build an MB2-C request.
type MB2CPeer struct {
	OriginHost       datatype.DiameterIdentity
	OriginRealm      datatype.DiameterIdentity
	DestinationHost  datatype.DiameterIdentity
	DestinationRealm datatype.DiameterIdentity
}

// NewGCSActionRequest creates a GCS-Action-Request (GAR) for the MB2-C
// application. The caller can append one of the GCS action AVPs, such as
// TMGI-Allocation-Request or TMGI-Deallocation-Request.
func NewGCSActionRequest(sessionID string, peer MB2CPeer, dictionary *dict.Parser) (*Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if peer.OriginHost == "" {
		return nil, fmt.Errorf("origin host is required")
	}
	if peer.OriginRealm == "" {
		return nil, fmt.Errorf("origin realm is required")
	}
	if peer.DestinationRealm == "" {
		return nil, fmt.Errorf("destination realm is required")
	}

	m := NewRequest(GCSAction, MB2C_APP_ID, dictionary)
	m.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String(sessionID))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(MB2C_APP_ID))
	m.NewAVP(avp.AuthSessionState, avp.Mbit, 0, AuthSessionStateNoStateMaintained)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, peer.OriginHost)
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, peer.OriginRealm)
	if peer.DestinationHost != "" {
		m.NewAVP(avp.DestinationHost, avp.Mbit, 0, peer.DestinationHost)
	}
	m.NewAVP(avp.DestinationRealm, avp.Mbit, 0, peer.DestinationRealm)

	return m, nil
}

// NewSupportedFeaturesAVP builds a 3GPP Supported-Features grouped AVP.
func NewSupportedFeaturesAVP(vendorID, featureListID, featureList uint32) *AVP {
	return NewAVP(avp.SupportedFeatures, avp.Mbit|avp.Vbit, TGPP_VENDOR_ID, &GroupedAVP{
		AVP: []*AVP{
			NewAVP(avp.VendorID, avp.Mbit, 0, datatype.Unsigned32(vendorID)),
			NewAVP(avp.FeatureListID, avp.Mbit|avp.Vbit, TGPP_VENDOR_ID, datatype.Unsigned32(featureListID)),
			NewAVP(avp.FeatureList, avp.Mbit|avp.Vbit, TGPP_VENDOR_ID, datatype.Unsigned32(featureList)),
		},
	})
}

// NewTMGIAllocationRequestAVP builds TMGI-Allocation-Request.
func NewTMGIAllocationRequestAVP(tmgiNumber uint32, tmgi ...datatype.OctetString) *AVP {
	g := &GroupedAVP{
		AVP: []*AVP{
			NewAVP(avp.TMGINumber, avp.Mbit|avp.Vbit, TGPP_VENDOR_ID, datatype.Unsigned32(tmgiNumber)),
		},
	}
	for _, t := range tmgi {
		g.AddAVP(NewAVP(avp.TMGI, avp.Vbit, TGPP_VENDOR_ID, t))
	}
	return NewAVP(avp.TMGIAllocationRequest, avp.Mbit|avp.Vbit, TGPP_VENDOR_ID, g)
}

// NewTMGIDeallocationRequestAVP builds TMGI-Deallocation-Request.
func NewTMGIDeallocationRequestAVP(tmgi ...datatype.OctetString) *AVP {
	g := &GroupedAVP{}
	for _, t := range tmgi {
		g.AddAVP(NewAVP(avp.TMGI, avp.Vbit, TGPP_VENDOR_ID, t))
	}
	return NewAVP(avp.TMGIDeallocationRequest, avp.Mbit|avp.Vbit, TGPP_VENDOR_ID, g)
}

// AddDefaultMB2CSupportedFeatures appends a Supported-Features AVP with a 3GPP
// Vendor-Id and zeroed feature-list values.
func AddDefaultMB2CSupportedFeatures(m *Message) {
	m.AddAVP(NewSupportedFeaturesAVP(TGPP_VENDOR_ID, 0, 0))
}

// AddTMGIAllocationRequest appends a TMGI-Allocation-Request to a GAR.
func AddTMGIAllocationRequest(m *Message, tmgiNumber uint32, tmgi ...datatype.OctetString) {
	m.AddAVP(NewTMGIAllocationRequestAVP(tmgiNumber, tmgi...))
}

// AddTMGIDeallocationRequest appends a TMGI-Deallocation-Request to a GAR.
func AddTMGIDeallocationRequest(m *Message, tmgi ...datatype.OctetString) {
	m.AddAVP(NewTMGIDeallocationRequestAVP(tmgi...))
}
