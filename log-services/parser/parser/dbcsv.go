package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
)

type dbcsvSection struct {
	name    string
	records []map[string]string
}

func parseCSV(file sourceFile) (core.ParsedLog, error) {
	rows, err := readDelimitedRows(file.data)
	if err != nil {
		return core.ParsedLog{}, fmt.Errorf("%w: csv read failed: %v", core.ErrParse, err)
	}

	if len(rows) < 2 {
		return core.ParsedLog{}, nil
	}

	headers := rows[0]
	records := make([]map[string]string, 0, len(rows)-1)

	for _, row := range rows[1:] {
		rec, err := rowToRecord(headers, row)
		if err != nil {
			return core.ParsedLog{}, fmt.Errorf("%w: csv row is malformed: %v", core.ErrParse, err)
		}
		if len(rec) > 0 {
			records = append(records, rec)
		}
	}

	return recordsToParsedLog(file.name, records)
}

func parseDBCSV(file sourceFile) (core.ParsedLog, error) {
	rows, err := readDelimitedRows(file.data)
	if err != nil {
		return core.ParsedLog{}, fmt.Errorf("%w: db_csv read failed: %v", core.ErrParse, err)
	}

	sections, err := splitDBCSVSections(rows)
	if err != nil {
		return core.ParsedLog{}, err
	}

	// Если файл оказался обычной CSV-таблицей без START_/END_,
	// пробуем обработать его как обычный CSV.
	if len(sections) == 0 {
		return parseCSV(file)
	}

	var parsed core.ParsedLog

	for _, section := range sections {
		sectionName := file.name + "/" + section.name
		sectionParsed, err := recordsToParsedLog(sectionName, section.records)
		if err != nil {
			return core.ParsedLog{}, err
		}

		parsed.Nodes = append(parsed.Nodes, sectionParsed.Nodes...)
		parsed.Ports = append(parsed.Ports, sectionParsed.Ports...)
		parsed.NodesInfo = append(parsed.NodesInfo, sectionParsed.NodesInfo...)
	}

	return parsed, nil
}

func readDelimitedRows(data []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	firstLine := firstNonEmptyLine(data)
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		reader.Comma = ';'
	}

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		clean := trimRow(row)
		if len(clean) > 0 {
			out = append(out, clean)
		}
	}

	return out, nil
}

func splitDBCSVSections(rows [][]string) ([]dbcsvSection, error) {
	var sections []dbcsvSection

	var currentName string
	var currentHeader []string
	var currentRecords []map[string]string

	flush := func() {
		if currentName == "" {
			currentHeader = nil
			currentRecords = nil
			return
		}

		if len(currentRecords) > 0 {
			sections = append(sections, dbcsvSection{
				name:    currentName,
				records: currentRecords,
			})
		}

		currentHeader = nil
		currentRecords = nil
	}

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		marker := normalizeSectionMarker(row[0])

		if strings.HasPrefix(marker, "START_") {
			flush()
			currentName = strings.TrimPrefix(marker, "START_")
			currentName = strings.ToLower(currentName)
			continue
		}

		if strings.HasPrefix(marker, "END_") {
			if currentName == "" {
				return nil, fmt.Errorf("%w: unexpected section end %q", core.ErrParse, row[0])
			}

			endName := strings.ToLower(strings.TrimPrefix(marker, "END_"))
			if endName != currentName {
				return nil, fmt.Errorf("%w: section end %q does not match start %q", core.ErrParse, endName, currentName)
			}

			flush()
			currentName = ""
			continue
		}

		if currentName == "" {
			continue
		}

		if isDBCSVMetaRow(row) {
			continue
		}

		if currentHeader == nil {
			currentHeader = row
			continue
		}

		rec, err := rowToRecord(currentHeader, row)
		if err != nil {
			return nil, fmt.Errorf("%w: section %s row is malformed: %v", core.ErrParse, currentName, err)
		}
		if len(rec) > 0 {
			currentRecords = append(currentRecords, rec)
		}
	}

	if currentName != "" {
		return nil, fmt.Errorf("%w: section %s is not closed", core.ErrParse, currentName)
	}

	flush()

	return sections, nil
}

func rowToRecord(headers []string, row []string) (map[string]string, error) {
	rec := make(map[string]string, len(headers))

	for i, header := range headers {
		if i >= len(row) {
			continue
		}

		key := strings.TrimSpace(header)
		value := strings.TrimSpace(row[i])

		if key == "" {
			continue
		}

		rec[key] = value
	}

	for i := len(headers); i < len(row); i++ {
		if strings.TrimSpace(row[i]) != "" {
			return nil, fmt.Errorf("extra value %q without header", row[i])
		}
	}

	return rec, nil
}

func trimRow(row []string) []string {
	out := make([]string, len(row))
	hasValue := false

	for i, value := range row {
		value = strings.TrimSpace(value)
		out[i] = value

		if value != "" {
			hasValue = true
		}
	}

	if !hasValue {
		return nil
	}

	return out
}

func normalizeSectionMarker(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = strings.ToUpper(value)

	return value
}

func isDBCSVMetaRow(row []string) bool {
	if len(row) == 0 {
		return true
	}

	first := strings.ToLower(strings.TrimSpace(row[0]))

	switch first {
	case "", "#", "comment", "comments":
		return true
	default:
		return strings.HasPrefix(first, "#")
	}
}
