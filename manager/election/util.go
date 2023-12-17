package election

import (
	"os"

	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"
)

const (
	unknownHostnameRandomNameLen = 12
)

// GenerateLeaderIdentity generates a string used for leader identity in the leader election
// process, this identity will be mostly the hostname, or if for some reason that cannot be fetched
// a random string.
func GenerateLeaderIdentity() string {
	hostname, err := os.Hostname()
	if err == nil {
		return clabernetesutilkubernetes.SafeConcatNameKubernetes(
			"clabernetes",
			hostname,
			clabernetesutil.RandomString(unknownHostnameRandomNameLen),
		)
	}

	return clabernetesutilkubernetes.SafeConcatNameKubernetes(
		"clabernetes",
		clabernetesutil.RandomString(unknownHostnameRandomNameLen),
	)
}
