package image

import (
	gocontext "context"
	"strings"

	"github.com/travis-ci/worker/config"
)

// EnvSelector implements Selector for environment-based mappings
type EnvSelector struct {
	c *config.ProviderConfig

	lookup map[string]string
}

// NewEnvSelector builds a new EnvSelector from the given *config.ProviderConfig
func NewEnvSelector(c *config.ProviderConfig) (*EnvSelector, error) {
	es := &EnvSelector{c: c}
	es.buildLookup()
	return es, nil
}

// LXD environment does not fully support env-based image selection.
// The minimal support stops working when the build request provides a group name
// that contains a `-`. Since the environment variables in bash do not support
// hypen based delimiting, we need to use a basic find-and-replace scheme to 
// implement the functionality. 

// The function works on the predicate that the group tag in builds will always
// have the `power-` keyword prefixed.
func modifyBuildGroup(group string) string {

	if strings.Contains(group, "group_power_") {
		return strings.Replace(group, "group_power_", "group_power-", 1)
	}

	return group
}

func (es *EnvSelector) buildLookup() {
	lookup := map[string]string{}

	es.c.Each(func(key, value string) {
		if strings.HasPrefix(key, "IMAGE_") {
			lookup[modifyBuildGroup(strings.ToLower(strings.Replace(key, "IMAGE_", "", -1)))] = value
		}
	})

	es.lookup = lookup
}

func (es *EnvSelector) Select(ctx gocontext.Context, params *Params) (string, error) {
	imageName := "default"

	for _, key := range es.buildCandidateKeys(params) {
		if key == "" {
			continue
		}

		if s, ok := es.lookup[key]; ok {
			imageName = s
			break
		}
	}

	// check for one level of indirection
	if selected, ok := es.lookup[imageName]; ok {
		return selected, nil
	}
	return imageName, nil
}

func (es *EnvSelector) buildCandidateKeys(params *Params) []string {
	fullKey := []string{}
	candidateKeys := []string{}

	hasDist := params.Dist != ""
	hasGroup := params.Group != ""
	hasOS := params.OS != ""

	if hasDist && hasGroup {
		candidateKeys = append(candidateKeys, "dist_"+params.Dist+"_group_"+params.Group)
		candidateKeys = append(candidateKeys, params.Dist+"_"+params.Group)
	}

	if hasDist {
		//candidateKeys = append(candidateKeys, "default_dist_"+params.Dist)
		candidateKeys = append(candidateKeys, "dist_"+params.Dist)
		candidateKeys = append(candidateKeys, params.Dist)
	}

	if hasGroup {
		//candidateKeys = append(candidateKeys, "default_group_"+params.Group)
		candidateKeys = append(candidateKeys, "group_"+params.Group)
		candidateKeys = append(candidateKeys, params.Group)
	}

	if hasOS {
		//candidateKeys = append(candidateKeys, "default_os_"+params.OS)
		candidateKeys = append(candidateKeys, "os_"+params.OS)
		candidateKeys = append(candidateKeys, params.OS)
	}

	return append([]string{strings.Join(fullKey, "_")}, candidateKeys...)
}
