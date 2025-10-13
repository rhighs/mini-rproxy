#!/usr/bin/env bash
curl --location 'http://localhost:8080/core/v2/visio/admin/Facility/GetGymKitSettings' \
    --header 'x-mwapps-appid: 9143e6d6-f36a-44e8-ae8c-4698ea897557' \
    --header 'x-mwapps-client: UnitySelfArtisCircuit' \
    --header 'x-equipment-context: '
    --header "x-mwapps-eqtoken: $1"
