// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

// textParams lists each string function's character-data arguments; a date passed there becomes text first, as CAST makes it.
var textParams = map[string][]int{
	"LEN": {0}, "DATALEN": nil, "UPPER": {0}, "LOWER": {0}, "LTRIM": {0, 1}, "RTRIM": {0, 1}, "TRIM": {AllArgs},
	"CONCAT": {AllArgs}, "CONCAT_WS": {AllArgs}, "REPLACE": {0, 1, 2}, "SUBSTRING": {0}, "QUOTENAME": {0, 1},
	"CHARINDEX": {0, 1}, "LEFT": {0}, "RIGHT": {0}, "REPLICATE": {0}, "STUFF": {0, 3}, "REVERSE": {0},
	"SPACE": nil, "ASCII": {0}, "CHAR": nil, "UNICODE": {0}, "NCHAR": nil, "PATINDEX": {0, 1},
	"TRANSLATE": {0, 1, 2}, "STR": nil, "STRING_ESCAPE": {0, 1}, "SOUNDEX": {0}, "DIFFERENCE": {0, 1},
	"FORMATMESSAGE": {0},
}

// AllArgs in a TextParams answer means every argument is character data.
const AllArgs = -1

// TextParams reports which arguments of a string function are character data; nil for any other function.
func TextParams(name string) []int { return textParams[name] }
