[![Discord](https://img.shields.io/discord/860500297297821756?style=flat-square&label=discord&logo=discord&color=00c9ff&labelColor=bec8d2)](https://discord.gg/vAyddtaEV9)
[![Go Report](https://img.shields.io/badge/go%20report-A%2B-blue?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://goreportcard.com/report/github.com/srl-labs/clabernetes)

# clabernetes a.k.a c9s

<p>
  <img src="https://gitlab.com/rdodin/pics/-/wikis/uploads/b5d611838fcb9c588b6311bccf11b954/c9s_logo1-upscale2x-white-tag+font-min__1_.png" width="300" align="left" alt="clabernetes"/>
  Love containerlab? Want containerlab, just distributed in a kubernetes cluster? Enter
  clabernetes -- containerlab + kubernetes. clabernetes is a kubernetes controller that deploys valid
  containerlab topologies into a kubernetes cluster.

  See [clabernetes docs](https://containerlab.dev/manual/clabernetes) for reference.
</p>

<br/>
<br/>

## Deploy

Deploying this chart is like deploying any other helm chart! The simplest case looks something like:

```bash
helm upgrade --install clabernetes oci://ghcr.io/srl-labs/clabernetes/clabernetes
```

You can select to install a specific version of the chart by adding the `--version` flag -- you can
find all the versions of the chart stored as a package on the projects GitHub page
[here](https://github.com/srl-labs/clabernetes/pkgs/container/clabernetes%2Fclabernetes).

## Values

As with most helm charts, this chart is configurable via values -- please refer to the charts
default values file for reference. You can find it
[here](https://github.com/srl-labs/clabernetes/blob/main/charts/clabernetes/values.yaml) or on the [Artifact Hub](https://artifacthub.io/packages/helm/clabernetes/clabernetes?modal=values).
