package analyzer

//
// Import graph analysis
//

// As soon as the algorithm detects that the `originalStart` node is reachable from other modules, it returns an error
func (self Analyzer) importGraphIsCyclicInner(originalStart string, start string, path []string) (outputPath []string, isCyclic bool) {
	// modules reachable from `start`
	module, found := self.modules[start]
	if !found {
		// this can occur if an import item is abandoned because it contains critical syntax errors
		return path, false
	}

	neighbors := module.ImportsModules

	for _, node := range neighbors {
		if node == originalStart {
			return append(path, node), true
		}
		if path, cyclic := self.importGraphIsCyclicInner(originalStart, node, append(path, node)); cyclic {
			return path, cyclic
		}
	}

	return path, false
}

func (self Analyzer) importGraphIsCyclic(start string) (outputPath []string, isCyclic bool) {
	return self.importGraphIsCyclicInner(start, start, []string{start})
}
