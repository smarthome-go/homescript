package runtime

func (self *Core) importItem(module string, toImport string) {
	val, found := self.Executor.GetBuiltinImport(module, toImport)
	if !found {
		panic("Every imported value is always found")
	}

	self.parent.globals.Mutex.Lock()
	defer self.parent.globals.Mutex.Unlock()
	// TODO: is this really legal
	self.parent.globals.Data[toImport] = val
}
