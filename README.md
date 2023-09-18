[![Discord](https://img.shields.io/discord/860500297297821756?style=flat-square&label=discord&logo=discord&color=00c9ff&labelColor=bec8d2)](https://discord.gg/vAyddtaEV9)
[![Go Report](https://img.shields.io/badge/go%20report-A%2B-blue?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://goreportcard.com/report/github.com/srl-labs/clabernetes)

# clabernetes a.k.a c9s

Love containerlab? Want containerlab, just distributed in a kubernetes cluster? Enter
clabernetes -- containerlab + kubernetes. clabernetes is a kubernetes controller that deploys valid
containerlab topologies into a kubernetes cluster.

See [clabernetes docs](http://containerlab.dev/manual/clabernetes) for reference.

## TODOs

- Make containerlab writing the deployment log to a file.
- Add ssh to the clab pods to enable local ssh access to the nodes.
- Watch docker process in container and restart if there is an issue
- Be very nice if we cached images maybe at the controller and/or somehow used cluster to pull
  image rather than having to pull every time container starts
- Generate kne binding for kne topos (and/or investigate doing that regardless of if topo is kne
  or not maybe... basically investigate just clab topo -> binding/testbed output so you can run
  fp tests against any topo)
