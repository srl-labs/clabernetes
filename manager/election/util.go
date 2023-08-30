package election

import (
	"os"

	clabernetesutil "github.com/srl-labs/clabernetes/util"
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
		return clabernetesutil.SafeConcatNameKubernetes(
			"clabernetes",
			hostname,
			clabernetesutil.RandomString(unknownHostnameRandomNameLen),
		)
	}

	return clabernetesutil.SafeConcatNameKubernetes(
		"clabernetes",
		clabernetesutil.RandomString(unknownHostnameRandomNameLen),
	)
}
