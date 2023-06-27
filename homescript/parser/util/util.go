package util

//
// Rune range helper functions
//

type RuneRange struct {
	min int
	max int
}

func IsRuneInRange(char rune, ranges ...RuneRange) bool {
	intChar := int(char)
	for _, ran := range ranges {
		if intChar >= ran.min && intChar <= ran.max {
			return true
		}
	}
	return false
}

func IsDigit(char rune) bool      { return IsRuneInRange(char, RuneRange{min: 48, max: 57}) }
func IsOctalDigit(char rune) bool { return IsRuneInRange(char, RuneRange{min: 48, max: 55}) }
func IsHexDigit(char rune) bool {
	return IsRuneInRange(char,
		RuneRange{min: 48, max: 57},
		RuneRange{min: 65, max: 70},
		RuneRange{min: 97, max: 102},
	)
}
func IsLetter(char rune) bool {
	return IsRuneInRange(
		char,
		RuneRange{min: 65, max: 90},  // capital letters
		RuneRange{min: 97, max: 122}, // lowercase letters
		RuneRange{min: 95, max: 95},  // underscore
	)
}

func IsIdent(test string) bool {
	if len(test) == 0 {
		return false
	}

	if !IsLetter(rune(test[0])) {
		return false
	}

	for i := 1; i < len(test); i++ {
		if !IsDigit(rune(test[i])) || !IsLetter(rune(test[i])) {
			return false
		}
	}

	return true
}
