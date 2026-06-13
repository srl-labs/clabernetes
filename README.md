[![Discord](https://img.shields.io/discord/860500297297821756?style=flat-square&label=discord&logo=discord&color=00c9ff&labelColor=bec8d2)](https://discord.gg/vAyddtaEV9)
[![Go Report](https://img.shields.io/badge/go%20report-A%2B-blue?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://goreportcard.com/report/github.com/srl-labs/clabernetes)

# clabernetes a.k.a c9s

<p>
  <img src="https://gitlab.com/rdodin/pics/-/wikis/uploads/b5d611838fcb9c588b6311bccf11b954/c9s_logo1-upscale2x-white-tag+font-min__1_.png" width="200" align="left" alt="clabernetes"/>
  Love containerlab? Want containerlab, just distributed in a kubernetes cluster? Enter
  clabernetes -- containerlab + kubernetes. clabernetes is a kubernetes controller that deploys valid
  containerlab topologies into a kubernetes cluster.

  See [clabernetes docs](https://containerlab.dev/manual/clabernetes) for reference.
</p>

<br clear="left"/>

## Try c9s

You can launch a disposable KinD cluster, install the published clabernetes Helm chart, and apply a
sample SR Linux plus multitool topology with:

```bash
make try-c9s
```

The target requires Docker and creates a single-node KinD cluster by default. It writes a KinD
config with a fixed UI host port mapping, installs MetalLB, and prints access endpoints:

```text
UI:                http://localhost:3000
SR Linux SSH:      ssh admin@<load-balancer-ip>
SR Linux gNMI:     <load-balancer-ip>:57400
SR Linux NETCONF:  <load-balancer-ip>:830
```

If KinD, kubectl, or Helm are not installed, it downloads local copies under
`build/try-c9s/bin`.

SR Linux management access uses the clabernetes LoadBalancer service
directly.

Clean up the sample resources and the KinD cluster with:

```bash
make try-c9s-clean
```
