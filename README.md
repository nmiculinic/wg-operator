# wg-operator 

This project aim to dynamically reconfigure wireguard on the fly for the cluster nodes. 

# QuickStart

See `/deploy` folder. Apply CRDs, that is under `/deploy/crds`. Example servers/clients are under `/deploy/servers` and `/deploy/clients`. Recommended deployment is also provided under `/deploy`

## Goals

* [x] Basic client-server VPN paradigm
* [ ] Implement IPtables masqerading for out of VPN IPs
* [ ] Highly scalable for clients (i.e. supporting 1000+ clients with minimal resource usage on client side)
* [ ] Medium dynamic network topology changes, wireguard setting & nodes won't change too often
* [ ] Unit test coverage + CI for config generation
* [ ] End2end test within CI
* [ ] Support key rotation
* [ ] Have decent usage documentation 

## Non-goals

* support OpenVPN or other VPN providers
* install wireguard on the target machines/perform upgrades. Use ansible or something else for it. Also look into https://github.com/KrakenSystems/wg-cni
