package main

// func main2() {
// 	programRaw, err := os.ReadFile(os.Args[1])
// 	if err != nil {
// 		panic(fmt.Sprintf("Could not read file `%s`: %s", os.Args[1], err.Error()))
// 	}
// 	program := string(programRaw)
// 	filename := strings.Split(os.Args[1], ".")[0]
//
// 	analyzed, diagnostics, syntaxErrors := homescript.Analyze(homescript.InputProgram{
// 		ProgramText: program,
// 		Filename:    filename,
// 	}, homescript.TestingAnalyzerScopeAdditions(), homescript.TestingAnalyzerHost{})
//
// 	if len(syntaxErrors) != 0 {
// 		for _, syntaxErr := range syntaxErrors {
// 			fmt.Printf("Reading: %s...\n", syntaxErr.Span.Filename)
//
// 			file, err := os.ReadFile(fmt.Sprintf("%s.hms", syntaxErr.Span.Filename))
// 			if err != nil {
// 				panic(err.Error())
// 			}
//
// 			fmt.Println(syntaxErr.Display(string(file)))
// 		}
// 		os.Exit(2)
// 	}
//
// 	abort := false
// 	fmt.Println("=== DIAGNOSTICS ===")
// 	for _, item := range diagnostics {
// 		if item.Level == diagnostic.DiagnosticLevelError {
// 			abort = true
// 		}
//
// 		fmt.Printf("Reading: %s...\n", item.Span.Filename)
//
// 		file, err := os.ReadFile(fmt.Sprintf("%s.hms", item.Span.Filename))
// 		if err != nil {
// 			panic(fmt.Sprintf("Could not read file `%s`: %s\n%s | %v", item.Span.Filename, err.Error(), item.Message, item.Span))
// 		}
//
// 		fmt.Println(item.Display(string(file)))
// 	}
//
// 	if abort {
// 		os.Exit(1)
// 	}
//
// 	fmt.Println("=== ANALYZED ===")
// 	for name, module := range analyzed {
// 		fmt.Printf("=== MODULE: %s ===\n", name)
// 		fmt.Println(module)
// 	}
//
// 	if os.Args[2] == "fuzz" {
// 		const passes = 4
// 		const seed = 42
// 		const passLimit = 1000
// 		const terminateAfterMinFound = 200 // 0 is unlimited
//
// 		outputDir := os.Args[3]
// 		if err := os.MkdirAll(outputDir, 0755); err != nil {
// 			panic(err.Error())
// 		}
//
// 		gen := fuzzer.NewGenerator(
// 			analyzed[filename],
// 			func(tree ast.AnalyzedProgram, treeString string, hashSum string) error {
// 				path := fmt.Sprintf("%s/%s.hms", outputDir, hashSum)
// 				return os.WriteFile(path, []byte(treeString), 0755)
// 			},
// 			seed,
// 			passes,
// 			terminateAfterMinFound,
// 			passLimit,
// 		)
// 		gen.Gen()
//
// 		return
// 	}
//
// 	if os.Args[2] == "vm" {
// 		homescript.TestingRunVm(analyzed, filename, true)
// 		return
// 	}
// 	if os.Args[2] == "tree" {
// 		homescript.TestingRunInterpreter(analyzed, filename)
// 		return
// 	}
//
// 	if os.Args[2] == "both" {
// 		homescript.TestingRunVm(analyzed, filename, true)
// 		homescript.TestingRunInterpreter(analyzed, filename)
// 		return
// 	}
//
// 	panic(fmt.Sprintf("Illegal run command `%s`", os.Args[2]))
// }
