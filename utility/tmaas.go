// Copyright 2020 Mavenir
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utility

import (
	"cim/types"
	"encoding/json"
	"fmt"
)

type TmaaSAnnotations struct {
	VendorID      string `json:"vendorId"`
	XGVelaID      string `json:"xgvelaId"`
	NfClass       string `json:"nfClass"`
	NfType        string `json:"nfType"`
	NfID          string `json:"nfId"`
	NfServiceID   string `json:"nfServiceId"`
	NfServiceType string `json:"nfServiceType"`
}

func GetAnnotations() (*TmaaSAnnotations, error) {

	var tmaasAnns *TmaaSAnnotations

	tmaasVals := types.PodDetails.Annotations["xgvela.com/tmaas"]
	if tmaasVals == "" {
		return nil, fmt.Errorf("tmaas annotation is not present")
	}

	err := json.Unmarshal([]byte(tmaasVals), &tmaasAnns)
	if err != nil {
		return nil, fmt.Errorf("Annotation spec unmarshal fail %s", err.Error())
	}

	return tmaasAnns, nil
}
