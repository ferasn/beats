// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package http

import (
	"bytes"
	"encoding/xml"
	"strings"
)

type XmlRequest struct {
	Initiator string
	MsgCode   string
	RqUID     string
}

type XmlResponse struct {
	Initiator      string
	MsgCode        string
	RqUID          string
	StatusCode     string
	StatusDesc     string
	StatusSeverity string
}

type SoapRequest struct {
	Initiator      string
	MsgCode        string
	RqUID          string
	UUID           string
	ServiceMsgType string
	SoapService    string
}

type SoapResponse struct {
	IsFault        bool
	FaultType      string
	FaultReason    string
	StatusCode     string
	StatusDesc     string
	StatusSeverity string
}

type SadadRequest struct {
	XmlReq  *XmlRequest
	SoapReq *SoapRequest
}

type SadadResponse struct {
	XmlRes  *XmlResponse
	SoapRes *SoapResponse
}

func NewXmlRequest() *XmlRequest {
	return &XmlRequest{Initiator: "None", MsgCode: "None", RqUID: "None"}
}

func NewXmlResponse() *XmlResponse {
	return &XmlResponse{StatusCode: "None", StatusDesc: "None", StatusSeverity: "None"}
}

func NewSoapRequest() *SoapRequest {
	return &SoapRequest{Initiator: "None", MsgCode: "None", RqUID: "None", ServiceMsgType: "None", SoapService: "None"}
}

func NewSoapResponse() *SoapResponse {
	return &SoapResponse{IsFault: false, FaultType: "None", FaultReason: "None", StatusCode: "None", StatusDesc: "None", StatusSeverity: "None"}
}

func readCharData(d *xml.Decoder) string {
	text := "None"
	t, err := d.Token()
	if err == nil {
		if charData, ok := t.(xml.CharData); ok {
			text = string(charData)
		}
	}
	return text
}

func parseXmlRequest(d *xml.Decoder) *XmlRequest {
	xmlReq := NewXmlRequest()
	for {
		t, err := d.Token()

		if err != nil {
			return xmlReq
		}

		switch et := t.(type) {
		case xml.StartElement:
			if et.Name.Local == "Sender" {
				xmlReq.Initiator = readCharData(d)
			} else if et.Name.Local == "MsgCode" {
				xmlReq.MsgCode = readCharData(d)
			} else if et.Name.Local == "RqUID" {
				xmlReq.RqUID = readCharData(d)
			}
		}
	}
}

func parseXmlResponse(d *xml.Decoder) *XmlResponse {
	//These flags to make sure that we are getting only header level status code and not record level
	statusCodeDone := false
	statusDescDone := false
	statusSeverityDone := false

	xmlRes := NewXmlResponse()
	for {
		t, err := d.Token()

		if err != nil {
			return xmlRes
		}

		switch et := t.(type) {
		case xml.StartElement:
			if et.Name.Local == "Receiver" {
				xmlRes.Initiator = readCharData(d)
			} else if et.Name.Local == "MsgCode" {
				xmlRes.MsgCode = readCharData(d)
				if len(xmlRes.MsgCode) > 1 {
					xmlRes.MsgCode = string(xmlRes.MsgCode[:len(xmlRes.MsgCode)-1]) + "Q"
				}
			} else if et.Name.Local == "RqUID" {
				xmlRes.RqUID = readCharData(d)
			} else if et.Name.Local == "StatusCode" && !statusCodeDone {
				xmlRes.StatusCode = readCharData(d)
				statusCodeDone = true
			} else if et.Name.Local == "ShortDesc" && !statusDescDone {
				xmlRes.StatusDesc = readCharData(d)
				statusDescDone = true
			} else if et.Name.Local == "Severity" && !statusSeverityDone {
				xmlRes.StatusSeverity = readCharData(d)
				statusSeverityDone = true
			}
		}
	}
}

func parseSoapRequest(d *xml.Decoder) *SoapRequest {
	soapReq := NewSoapRequest()

	//Look for SOAP Body.  SoapService is the next element after Body
	bodyFound := false
	soapServiceFound := false
	for !soapServiceFound {
		t, err := d.Token()
		if err != nil {
			return soapReq
		}

		switch et := t.(type) {
		case xml.StartElement:
			if et.Name.Local == "MessageId" { //This is in bulk upload
				soapReq.RqUID = readCharData(d)
			} else if et.Name.Local == "ConversationId" { //This is in bulk upload
				soapReq.UUID = readCharData(d)
			} else if et.Name.Local == "PartyId" { //This is in bulk upload
				partyID := readCharData(d) //This is a dirty workaround to quickly get the partyID that belongs to FROM section
				if !strings.HasPrefix(partyID, "SADAD") {
					soapReq.Initiator = partyID
				}
			} else if et.Name.Local == "Service" { //This is in bulk upload
				soapReq.MsgCode = readCharData(d)
			} else if et.Name.Local == "ServiceInitiatorKey" { //Some requests include MessageHeader section in SOAP Header
				soapReq.Initiator = readCharData(d)
			} else if et.Name.Local == "RqUID" {
				soapReq.RqUID = readCharData(d)
			} else if et.Name.Local == "UUID" {
				soapReq.UUID = readCharData(d)
			} else if et.Name.Local == "MsgCode" {
				soapReq.MsgCode = readCharData(d)
			} else if et.Name.Local == "ServiceMsgType" {
				soapReq.ServiceMsgType = readCharData(d)
			} else if et.Name.Local == "Body" {
				bodyFound = true
			} else if bodyFound {
				soapReq.SoapService = et.Name.Local
				soapServiceFound = true
			}
		}
	}

	for {
		t, err := d.Token()

		if err != nil {
			return soapReq
		}

		switch et := t.(type) {
		case xml.StartElement:
			if et.Name.Local == "ServiceInitiatorKey" { //This capture MessageHeader in SOAP Body
				soapReq.Initiator = readCharData(d)
			} else if et.Name.Local == "RqUID" {
				soapReq.RqUID = readCharData(d)
			} else if et.Name.Local == "UUID" {
				soapReq.UUID = readCharData(d)
			} else if et.Name.Local == "MsgCode" {
				soapReq.MsgCode = readCharData(d)
			} else if et.Name.Local == "ServiceMsgType" {
				soapReq.ServiceMsgType = readCharData(d)
			}
		}
	}
}

func parseSoapResponse(d *xml.Decoder) *SoapResponse {
	soapRes := NewSoapResponse()

	//These flags to make sure that we are getting only header level status code and not record level
	statusCodeDone := false
	statusDescDone := false
	statusSeverityDone := false

	for {
		t, err := d.Token()

		if err != nil {
			return soapRes
		}

		switch et := t.(type) {
		case xml.StartElement:
			if et.Name.Local == "StatusCode" && !statusCodeDone {
				soapRes.StatusCode = readCharData(d)
				statusCodeDone = true
			} else if et.Name.Local == "StatusDesc" && !statusDescDone {
				soapRes.StatusDesc = readCharData(d)
				statusDescDone = true
			} else if et.Name.Local == "Severity" && !statusSeverityDone {
				soapRes.StatusSeverity = readCharData(d)
				statusSeverityDone = true
			} else if et.Name.Local == "Fault" {
				soapRes.IsFault = true
			} else if et.Name.Local == "Code" {
				soapRes.StatusCode = readCharData(d)
			} else if et.Name.Local == "Type" {
				soapRes.FaultType = readCharData(d)
			} else if et.Name.Local == "Description" {
				soapRes.StatusDesc = readCharData(d)
			} else if et.Name.Local == "Reason" {
				soapRes.FaultReason = readCharData(d)
			} else if et.Name.Local == "Severity" {
				soapRes.StatusSeverity = readCharData(d)
			}
		}
	}
}

func ParseSadadReqMsg(msg []byte) *SadadRequest {
	request := &SadadRequest{}

	b := bytes.NewBuffer(msg)
	d := xml.NewDecoder(b)

	var elm xml.StartElement
	//Loop XML nodes until you get the root element. First node could a processing instruction or comment
	for {
		t, err := d.Token()
		if err != nil {
			return request
		}

		var ok bool
		elm, ok = t.(xml.StartElement)
		if ok {
			//Root element is reached
			break
		}
	}

	if elm.Name.Local == "Envelope" {
		//SOAP message
		request.SoapReq = parseSoapRequest(d)
	} else if elm.Name.Local == "SADAD" {
		//XML message
		request.XmlReq = parseXmlRequest(d)
	} else {
		//Add some logging
	}

	return request
}

func ParseSadadResMsg(msg []byte) *SadadResponse {
	response := &SadadResponse{}

	b := bytes.NewBuffer(msg)
	d := xml.NewDecoder(b)

	var elm xml.StartElement
	//Loop XML nodes until you get the root element. First node could a processing instruction or comment
	for {
		t, err := d.Token()
		if err != nil {
			return response
		}

		var ok bool
		elm, ok = t.(xml.StartElement)
		if ok {
			//Root element is reached
			break
		}
	}

	if elm.Name.Local == "Envelope" {
		//SOAP message
		response.SoapRes = parseSoapResponse(d)
	} else if elm.Name.Local == "SADAD" {
		//XML message
		response.XmlRes = parseXmlResponse(d)
	} else {
		//Add some logging
	}

	return response
}
