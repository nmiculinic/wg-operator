[![Build Status](https://gitlab.com/neven-miculinic/wg-operator/badges/master/pipeline.svg)](https://gitlab.com/neven-miculinic/wg-operator/pipelines) [![GoDoc](https://godoc.org/github.com/KrakenSystems/wireguardctrl?status.svg)](https://godoc.org/github.com/KrakenSystems/wg-operator) [![Go Report Card](https://goreportcard.com/badge/github.com/KrakenSystems/wg-operator)](https://goreportcard.com/report/github.com/KrakenSystems/wg-operator)
# wg-operator

This project aim to dynamically reconfigure wireguard on the fly for the cluster nodes.

# QuickStart

See `/deploy` folder. Apply CRDs, that is under `/deploy/crds`. Example servers/clients are under `/deploy/servers` and `/deploy/clients`. Recommended deployment is also provided under `/deploy`

## Goals

* [x] Basic client-server VPN paradigm
* [ ] Implement IPtables masqerading for out of VPN IPs --> use preUp/postDown for now, and wg-quick or wg-quick-go to run them at system boot.
* [ ] Highly scalable for clients (i.e. supporting 1000+ clients with minimal resource usage on client side). For mostly static topologies this should be quite performant.
    * [x] update coalescing --> implemented via 200ms coalescing time window
    * [ ] error exponential backoff --> Not implemented, on error we retry every 5 seconds
    * [ ] client query only myself --> partially implemeted, informer cache is fetching all client changes, but update is triggered only for myself
* [ ] Implement per server interface for clients -- allows custom routing to operate on top of wireguard (e.g. OSPF/BGP)
* [x] Medium dynamic network topology changes, wireguard setting & nodes won't change too often
* [ ] Unit test coverage + CI for config generation
* [ ] End2end test within CI
* [ ] Support key rotation
* [ ] Have decent usage documentation

## Non-goals

* support OpenVPN or other VPN providers
* install wireguard on the target machines/perform upgrades. Use ansible or something else for it. Also look into https://github.com/KrakenSystems/wg-cni

# Docker images registy, automatically built via CI pipeline

It's located at:

* https://gitlab.com/neven-miculinic/wg-operator/container_registry

Per branch images:

registry.gitlab.com/neven-miculinic/wg-operator:<branch-name>-<arch>
registry.gitlab.com/neven-miculinic/wg-operator:<branch-name>-<short commit hash>-<arch>

Example:
* registry.gitlab.com/neven-miculinic/wg-operator:master-6b18ddbf-amd64
* registry.gitlab.com/neven-miculinic/wg-operator:master-6b18ddbf-arm32v7
* registry.gitlab.com/neven-miculinic/wg-operator:master-6b18ddbf-arm64v8
* registry.gitlab.com/neven-miculinic/wg-operator:master-amd64
* registry.gitlab.com/neven-miculinic/wg-operator:master-arm32v7
* registry.gitlab.com/neven-miculinic/wg-operator:master-arm64v8
