// Copyright 2016 - 2019 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to
// and read from XLSX files. Support reads and writes XLSX file generated by
// Microsoft Excel™ 2007 and later. Support save file without losing original
// charts of XLSX. This library needs Go version 1.10 or later.

package excelize

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

// ReadZipReader can be used to read an XLSX in memory without touching the
// filesystem.
func ReadZipReader(r *zip.Reader) (map[string][]byte, int, error) {
	fileList := make(map[string][]byte)
	worksheets := 0
	for _, v := range r.File {
		fileList[v.Name] = readFile(v)
		if len(v.Name) > 18 {
			if v.Name[0:19] == "xl/worksheets/sheet" {
				worksheets++
			}
		}
	}
	return fileList, worksheets, nil
}

// readXML provides a function to read XML content as string.
func (f *File) readXML(name string) []byte {
	if content, ok := f.XLSX[name]; ok {
		return content
	}
	return []byte{}
}

// saveFileList provides a function to update given file content in file list
// of XLSX.
func (f *File) saveFileList(name string, content []byte) {
	newContent := make([]byte, 0, len(XMLHeader)+len(content))
	newContent = append(newContent, []byte(XMLHeader)...)
	newContent = append(newContent, content...)
	f.XLSX[name] = newContent
}

// Read file content as string in a archive file.
func readFile(file *zip.File) []byte {
	rc, err := file.Open()
	if err != nil {
		log.Panic(err)
	}
	buff := bytes.NewBuffer(nil)
	_, _ = io.Copy(buff, rc)
	rc.Close()
	return buff.Bytes()
}

// SplitCellName splits cell name to column name and row number.
//
// Example:
//
//     excelize.SplitCellName("AK74") // return "AK", 74, nil
//
func SplitCellName(cell string) (string, int, error) {
	alpha := func(r rune) bool {
		return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
	}

	if strings.IndexFunc(cell, alpha) == 0 {
		i := strings.LastIndexFunc(cell, alpha)
		if i >= 0 && i < len(cell)-1 {
			col, rowstr := cell[:i+1], cell[i+1:]
			if row, err := strconv.Atoi(rowstr); err == nil && row > 0 {
				return col, row, nil
			}
		}
	}
	return "", -1, newInvalidCellNameError(cell)
}

// JoinCellName joins cell name from column name and row number.
func JoinCellName(col string, row int) (string, error) {
	normCol := strings.Map(func(rune rune) rune {
		switch {
		case 'A' <= rune && rune <= 'Z':
			return rune
		case 'a' <= rune && rune <= 'z':
			return rune - 32
		}
		return -1
	}, col)
	if len(col) == 0 || len(col) != len(normCol) {
		return "", newInvalidColumnNameError(col)
	}
	if row < 1 {
		return "", newInvalidRowNumberError(row)
	}
	return fmt.Sprintf("%s%d", normCol, row), nil
}

// ColumnNameToNumber provides a function to convert Excel sheet column name
// to int. Column name case insensitive. The function returns an error if
// column name incorrect.
//
// Example:
//
//     excelize.ColumnNameToNumber("AK") // returns 37, nil
//
func ColumnNameToNumber(name string) (int, error) {
	if len(name) == 0 {
		return -1, newInvalidColumnNameError(name)
	}
	col := 0
	multi := 1
	for i := len(name) - 1; i >= 0; i-- {
		r := name[i]
		if r >= 'A' && r <= 'Z' {
			col += int(r-'A'+1) * multi
		} else if r >= 'a' && r <= 'z' {
			col += int(r-'a'+1) * multi
		} else {
			return -1, newInvalidColumnNameError(name)
		}
		multi *= 26
	}
	return col, nil
}

// ColumnNumberToName provides a function to convert the integer to Excel
// sheet column title.
//
// Example:
//
//     excelize.ColumnNumberToName(37) // returns "AK", nil
//
func ColumnNumberToName(num int) (string, error) {
	if num < 1 {
		return "", fmt.Errorf("incorrect column number %d", num)
	}
	var col string
	for num > 0 {
		col = string((num-1)%26+65) + col
		num = (num - 1) / 26
	}
	return col, nil
}

// CellNameToCoordinates converts alphanumeric cell name to [X, Y] coordinates
// or returns an error.
//
// Example:
//
//    CellCoordinates("A1") // returns 1, 1, nil
//    CellCoordinates("Z3") // returns 26, 3, nil
//
func CellNameToCoordinates(cell string) (int, int, error) {
	const msg = "cannot convert cell %q to coordinates: %v"

	colname, row, err := SplitCellName(cell)
	if err != nil {
		return -1, -1, fmt.Errorf(msg, cell, err)
	}

	col, err := ColumnNameToNumber(colname)
	if err != nil {
		return -1, -1, fmt.Errorf(msg, cell, err)
	}

	return col, row, nil
}

// CoordinatesToCellName converts [X, Y] coordinates to alpha-numeric cell
// name or returns an error.
//
// Example:
//
//    CoordinatesToCellName(1, 1) // returns "A1", nil
//
func CoordinatesToCellName(col, row int) (string, error) {
	if col < 1 || row < 1 {
		return "", fmt.Errorf("invalid cell coordinates [%d, %d]", col, row)
	}
	colname, err := ColumnNumberToName(col)
	if err != nil {
		return "", fmt.Errorf("invalid cell coordinates [%d, %d]: %v", col, row, err)
	}
	return fmt.Sprintf("%s%d", colname, row), nil
}

// boolPtr returns a pointer to a bool with the given value.
func boolPtr(b bool) *bool { return &b }

// defaultTrue returns true if b is nil, or the pointed value.
func defaultTrue(b *bool) bool {
	if b == nil {
		return true
	}
	return *b
}

// parseFormatSet provides a method to convert format string to []byte and
// handle empty string.
func parseFormatSet(formatSet string) []byte {
	if formatSet != "" {
		return []byte(formatSet)
	}
	return []byte("{}")
}

// namespaceStrictToTransitional provides a method to convert Strict and
// Transitional namespaces.
func namespaceStrictToTransitional(content []byte) []byte {
	var namespaceTranslationDic = map[string]string{
		StrictSourceRelationship:         SourceRelationship,
		StrictSourceRelationshipChart:    SourceRelationshipChart,
		StrictSourceRelationshipComments: SourceRelationshipComments,
		StrictSourceRelationshipImage:    SourceRelationshipImage,
		StrictNameSpaceSpreadSheet:       NameSpaceSpreadSheet,
	}
	for s, n := range namespaceTranslationDic {
		content = bytes.Replace(content, []byte(s), []byte(n), -1)
	}
	return content
}

// genSheetPasswd provides a method to generate password for worksheet
// protection by given plaintext. When an Excel sheet is being protected with
// a password, a 16-bit (two byte) long hash is generated. To verify a
// password, it is compared to the hash. Obviously, if the input data volume
// is great, numerous passwords will match the same hash. Here is the
// algorithm to create the hash value:
//
// take the ASCII values of all characters shift left the first character 1 bit,
// the second 2 bits and so on (use only the lower 15 bits and rotate all higher bits,
// the highest bit of the 16-bit value is always 0 [signed short])
// XOR all these values
// XOR the count of characters
// XOR the constant 0xCE4B
func genSheetPasswd(plaintext string) string {
	var password int64 = 0x0000
	var charPos uint = 1
	for _, v := range plaintext {
		value := int64(v) << charPos
		charPos++
		rotatedBits := value >> 15 // rotated bits beyond bit 15
		value &= 0x7fff            // first 15 bits
		password ^= (value | rotatedBits)
	}
	password ^= int64(len(plaintext))
	password ^= 0xCE4B
	return strings.ToUpper(strconv.FormatInt(password, 16))
}
