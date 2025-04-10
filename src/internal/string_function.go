package internal

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yorukot/superfile/src/internal/common"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/term/ansi"
)

func truncateText(text string, maxChars int, talis string) string {
	truncatedText := ansi.Truncate(text, maxChars-len(talis), "")
	if text != truncatedText {
		return truncatedText + talis
	}

	return text
}

func truncateTextBeginning(text string, maxChars int, talis string) string {
	if ansi.StringWidth(text) <= maxChars {
		return text
	}

	truncatedRunes := []rune(text)

	truncatedWidth := ansi.StringWidth(string(truncatedRunes))

	for truncatedWidth > maxChars {
		truncatedRunes = truncatedRunes[1:]
		truncatedWidth = ansi.StringWidth(string(truncatedRunes))
	}

	if len(truncatedRunes) > len(talis) {
		truncatedRunes = append([]rune(talis), truncatedRunes[len(talis):]...)
	}

	return string(truncatedRunes)
}

func truncateMiddleText(text string, maxChars int, talis string) string {
	if utf8.RuneCountInString(text) <= maxChars {
		return text
	}

	halfEllipsisLength := (maxChars - 3) / 2

	truncatedText := text[:halfEllipsisLength] + talis + text[utf8.RuneCountInString(text)-halfEllipsisLength:]

	return truncatedText
}

func prettierName(name string, width int, isDir bool, isSelected bool, bgColor lipgloss.Color) string {
	style := getElementIcon(name, isDir)
	if isSelected {
		return common.StringColorRender(lipgloss.Color(style.Color), bgColor).
			Background(bgColor).
			Render(style.Icon+" ") +
			common.FilePanelItemSelectedStyle.
				Render(truncateText(name, width, "..."))
	}
	return common.StringColorRender(lipgloss.Color(style.Color), bgColor).
		Background(bgColor).
		Render(style.Icon+" ") +
		common.FilePanelStyle.Render(truncateText(name, width, "..."))
}

func prettierDirectoryPreviewName(name string, isDir bool, bgColor lipgloss.Color) string {
	style := getElementIcon(name, isDir)
	return common.StringColorRender(lipgloss.Color(style.Color), bgColor).
		Background(bgColor).
		Render(style.Icon+" ") +
		common.FilePanelStyle.Render(name)
}

func clipboardPrettierName(name string, width int, isDir bool, isSelected bool) string {
	style := getElementIcon(name, isDir)
	if isSelected {
		return common.StringColorRender(lipgloss.Color(style.Color), common.FooterBGColor).
			Background(common.FooterBGColor).
			Render(style.Icon+" ") +
			common.FilePanelItemSelectedStyle.Render(truncateTextBeginning(name, width, "..."))
	}
	return common.StringColorRender(lipgloss.Color(style.Color), common.FooterBGColor).
		Background(common.FooterBGColor).
		Render(style.Icon+" ") +
		common.FilePanelStyle.Render(truncateTextBeginning(name, width, "..."))
}

func fileNameWithoutExtension(fileName string) string {
	for {
		pos := strings.LastIndexByte(fileName, '.')
		if pos <= 0 {
			break
		}
		fileName = fileName[:pos]
	}
	return fileName
}

func formatFileSize(size int64) string {
	if size == 0 {
		return "0B"
	}

	unitsDec := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	unitsBin := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

	// Todo : Remove duplication here
	if common.Config.FileSizeUseSI {
		unitIndex := int(math.Floor(math.Log(float64(size)) / math.Log(1000)))
		adjustedSize := float64(size) / math.Pow(1000, float64(unitIndex))
		return fmt.Sprintf("%.2f %s", adjustedSize, unitsDec[unitIndex])
	}
	unitIndex := int(math.Floor(math.Log(float64(size)) / math.Log(1024)))
	adjustedSize := float64(size) / math.Pow(1024, float64(unitIndex))
	return fmt.Sprintf("%.2f %s", adjustedSize, unitsBin[unitIndex])
}

// Truncate line lengths and keep ANSI
func checkAndTruncateLineLengths(text string, maxLength int) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder

	for _, line := range lines {
		// Replace tabs with spaces
		expandedLine := strings.ReplaceAll(line, "\t", strings.Repeat(" ", 4))
		truncatedLine := ansi.Truncate(expandedLine, maxLength, "")
		result.WriteString(truncatedLine + "\n")
	}

	finalResult := strings.TrimRight(result.String(), "\n")

	return finalResult
}

// Separated this out out for easy testing
func isBufferPrintable(buffer []byte) bool {
	for _, b := range buffer {
		// This will also handle b==0
		if !unicode.IsPrint(rune(b)) && !unicode.IsSpace(rune(b)) {
			return false
		}
	}
	return true
}

// isExensionExtractable checks if a string is a valid compressed archive file extension.
func isExensionExtractable(ext string) bool {
	// Extensions based on the types that package: `xtractr` `ExtractFile` function handles.
	validExtensions := map[string]struct{}{
		".zip":     {},
		".bz":      {},
		".gz":      {},
		".iso":     {},
		".rar":     {},
		".7z":      {},
		".tar":     {},
		".tar.gz":  {},
		".tar.bz2": {},
	}
	_, exists := validExtensions[strings.ToLower(ext)]
	return exists
}

// Check file is text file or not
func isTextFile(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	cnt, err := reader.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	return isBufferPrintable(buffer[:cnt]), nil
}

// Although some characters like `\x0b`(vertical tab) are printable,
// previewing them breaks the layout.
// So, among the "non-graphic" printable characters, we only need \n and \t
// Space and NBSP are already considered graphic by unicode.
func makePrintable(line string) string {
	var sb strings.Builder
	// This has to be looped byte-wise, looping it rune-wise
	// or by using strings.Map would cause issues with strings like
	// "(NBSP)\xa0"
	for i := range len(line) {
		r := rune(line[i])
		if unicode.IsGraphic(r) || r == rune('\t') || r == rune('\n') {
			sb.WriteByte(line[i])
		}
	}
	return sb.String()
}
