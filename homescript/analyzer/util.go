package analyzer

func (self *Analyzer) pushScope() {
	(*self.currentModule).Scopes = append((*self.currentModule).Scopes, newScope())
}

func (self *Analyzer) setCurrentModule(name string) {
	self.currentModuleName = name
	self.currentModule = self.modules[name]
}
