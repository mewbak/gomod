package depgraph

import (
	"io/ioutil"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetDepGraph should be called from within a Go module. It will return the dependency
// graph for this module.
func GetDepGraph(logger *logrus.Logger) (*DepGraph, error) {
	logger.Debug("Creating dependency graph.")
	main, modules, err := getSelectedModules(logger)
	if err != nil {
		return nil, err
	}

	rawDeps, err := runCommand(logger, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	graph := &DepGraph{
		logger: logger,
		module: main.Path,
		nodes:  map[string]*Node{},
	}
	if graph.logger == nil {
		graph.logger = logrus.New()
		graph.logger.SetOutput(ioutil.Discard)
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(rawDeps)), "\n") {
		logger.Debugf("Parsing dependency: %s", dep)

		depContent := depRE.FindStringSubmatch(dep)
		if len(depContent) == 0 {
			logger.Warnf("Ill-formed line in 'go mod graph' output: %s", dep)
			continue
		}

		beginNodeName, beginVersion := depContent[1], depContent[2]
		endNodeName, endVersion := depContent[3], depContent[4]

		beginNode := graph.nodes[beginNodeName]
		if beginNode == nil {
			beginNode, err = createNewNode(beginNodeName, modules)
			if err != nil {
				return nil, err
			}
			graph.nodes[beginNodeName] = beginNode
			logger.Debugf("Created new node: %+v", beginNode)
		}
		endNode := graph.nodes[endNodeName]
		if endNode == nil {
			endNode, err = createNewNode(endNodeName, modules)
			if err != nil {
				return nil, err
			}
			graph.nodes[endNodeName] = endNode
			logger.Debugf("Created new node: %+v", endNode)
		}

		if len(beginNode.SelectedVersion()) != 0 && beginNode.module.Replace == nil && beginNode.SelectedVersion() != beginVersion {
			logger.Warnf("Encountered unexpected version %q for edge starting at node %q.", beginVersion, beginNodeName)
		}
		newDependency := &Dependency{
			begin:   beginNodeName,
			end:     endNodeName,
			version: endVersion,
		}
		beginNode.successors = append(beginNode.successors, newDependency)
		endNode.predecessors = append(endNode.predecessors, newDependency)
		logger.Debugf("Created new dependency: %+v", newDependency)
	}
	return graph, nil
}