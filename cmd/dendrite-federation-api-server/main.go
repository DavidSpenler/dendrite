// Copyright 2017 Vector Creations Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/matrix-org/dendrite/clientapi/producers"
	"github.com/matrix-org/dendrite/eduserver"
	"github.com/matrix-org/dendrite/eduserver/cache"
	"github.com/matrix-org/dendrite/federationapi"
	"github.com/matrix-org/dendrite/internal/basecomponent"
)

func main() {
	cfg := basecomponent.ParseFlags()
	base := basecomponent.NewBaseDendrite(cfg, "FederationAPI", true)
	defer base.Close() // nolint: errcheck

	accountDB := base.CreateAccountsDB()
	deviceDB := base.CreateDeviceDB()
	federation := base.CreateFederationClient()

	serverKeyAPI := base.ServerKeyAPIClient()
	keyRing := serverKeyAPI.KeyRing()

	fsAPI := base.FederationSenderHTTPClient()

	rsAPI := base.RoomserverHTTPClient()
	asAPI := base.AppserviceHTTPClient()
	rsAPI.SetFederationSenderAPI(fsAPI)
	eduInputAPI := eduserver.SetupEDUServerComponent(base, cache.New(), deviceDB)
	eduProducer := producers.NewEDUServerProducer(eduInputAPI)

	federationapi.SetupFederationAPIComponent(
		base, accountDB, deviceDB, federation, keyRing,
		rsAPI, asAPI, fsAPI, eduProducer,
	)

	base.SetupAndServeHTTP(string(base.Cfg.Bind.FederationAPI), string(base.Cfg.Listen.FederationAPI))

}
