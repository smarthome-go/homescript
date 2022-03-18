package homescript

func isDigit(char rune) bool {
	intChar := int(char)
	return intChar >= 48 && intChar <= 57
}

func isLetter(char rune) bool {
	intChar := int(char)
	return intChar >= 65 && intChar <= 90 || intChar >= 97 && intChar <= 122
}
