package filters

import (
	"strings"

	"github.com/blang/semver"
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

// TargetDependencies implements the `depgraph.Filter` interface. It removes any edges that are not
// part of a chain leading to one of the specified dependencies. If for a given dependency has a
// version set, we only keep edges that prevent the use of the dependency at that given version
// under the constraints of minimal version selection.
type TargetDependencies struct {
	Targets []*struct {
		Module  string
		Version string
	}
}

// SubGraph returns a copy of the dependency graph with all dependencies that are part of chains
// that need to be modified for the specified dependency to be set to a given target version
// annotated as such.
func (f *TargetDependencies) Apply(logger *logrus.Logger, graph *depgraph.DepGraph) *depgraph.DepGraph {
	if len(f.Targets) == 0 {
		return graph
	}

	keep := map[string]struct{}{}
	for _, dep := range f.Targets {
		keep = applyFilter(logger, graph, &targetDependencyFilter{
			module:  dep.Module,
			version: dep.Version,
		}, keep)
	}

	logger.Debug("Pruning the dependency graph of irrelevant paths.")
	subGraph := graph.DeepCopy()
	for _, dependency := range graph.Dependencies.List() {
		if _, ok := keep[dependency.Name()]; !ok {
			logger.Debugf("Pruning %q.", dependency.Name())
			subGraph.RemoveDependency(dependency.Name())
		}
	}
	return subGraph
}

type targetDependencyFilter struct {
	module  string
	version string
}

func applyFilter(
	logger *logrus.Logger,
	graph *depgraph.DepGraph,
	filter *targetDependencyFilter,
	keep map[string]struct{},
) map[string]struct{} {
	filterModule, ok := graph.GetDependency(filter.module)
	if !ok {
		return nil
	}

	if keep == nil {
		keep = map[string]struct{}{}
	}
	keep[filterModule.Name()] = struct{}{}

	logger.Debugf("Marking subgraph for dependency %q.", filter.module)
	if filter.version != "" {
		logger.Debugf("Only considering dependencies preventing use of version %q.", filter.version)
	}
	var todo []*depgraph.DependencyReference
	for _, predecessor := range filterModule.Predecessors.List() {
		if dependencyMatchesFilter(predecessor, filter) {
			todo = append(todo, predecessor)
			keep[predecessor.Name()] = struct{}{}
		}
	}

	for len(todo) > 0 {
		dependency := todo[0]
		for _, predecessor := range dependency.Predecessors.List() {
			if _, ok := keep[predecessor.Name()]; !ok {
				keep[predecessor.Name()] = struct{}{}
				todo = append(todo, predecessor)
			}
		}
		todo = todo[1:]
	}
	return keep
}

func dependencyMatchesFilter(dependency *depgraph.DependencyReference, filter *targetDependencyFilter) bool {
	if dependency.VersionConstraint == "" || filter.version == "" {
		return true
	}
	constraint := semver.MustParse(strings.TrimLeft(dependency.VersionConstraint, "v"))
	depVersion := semver.MustParse(strings.TrimLeft(filter.version, "v"))
	return constraint.GT(depVersion)
}
