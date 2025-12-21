package policy

import (
	"os"

	"github.com/davidahmann/relia_oss/internal/crypto"
	"gopkg.in/yaml.v3"
)

type LoadedPolicy struct {
	Policy Policy
	Hash   string
	Bytes  []byte
}

// LoadPolicy loads a YAML policy and computes its hash from raw bytes.
func LoadPolicy(path string) (LoadedPolicy, error) {
	// #nosec G304 -- path comes from operator-configured policy path.
	data, err := os.ReadFile(path)
	if err != nil {
		return LoadedPolicy{}, err
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return LoadedPolicy{}, err
	}

	return LoadedPolicy{
		Policy: p,
		Hash:   crypto.DigestWithPrefix(data),
		Bytes:  data,
	}, nil
}
