package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Config struct {
	Table     string
	Name      string
	SchemaSQL string
	SQLCYAML  string
	Module    string
	Overwrite bool
}

type Table struct {
	Schema  string
	Name    string
	Columns []Column
}

type Column struct {
	Name       string
	SQLType    string
	GoType     string
	PrimaryKey bool
	HasDefault bool
	IsSerial   bool
	NotNull    bool
	Insertable bool
}

type Domain struct {
	PackageName string
	TypeName    string
	Table       Table
	Columns     []Column
	Module      string
	ImportTime  bool
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.Table, "table", "", "table name, e.g. v1.notices")
	flag.StringVar(&cfg.Name, "name", "", "domain package name, e.g. notice")
	flag.StringVar(&cfg.SchemaSQL, "schema", "./ops/db/init.sql", "schema sql path")
	flag.StringVar(&cfg.SQLCYAML, "sqlc", "./sqlc.yaml", "sqlc yaml path")
	flag.StringVar(&cfg.Module, "module", "", "go module path")
	flag.BoolVar(&cfg.Overwrite, "overwrite", false, "overwrite generated files")
	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "generator:", err)
		os.Exit(1)
	}
}

func run(cfg Config) error {
	if cfg.Table == "" {
		return fmt.Errorf("-table is required")
	}

	module := cfg.Module
	if module == "" {
		var err error
		module, err = readModulePath("go.mod")
		if err != nil {
			return err
		}
	}

	sqlBytes, err := os.ReadFile(cfg.SchemaSQL)
	if err != nil {
		return err
	}

	table, err := findCreateTable(string(sqlBytes), cfg.Table)
	if err != nil {
		return err
	}

	domainName := cfg.Name
	if domainName == "" {
		domainName = singular(table.Name)
	}
	pkgName := packageName(domainName)
	if pkgName == "" {
		return fmt.Errorf("invalid domain name")
	}

	domain := buildDomain(table, domainName, pkgName, module)

	baseDir := filepath.Join("domain", domain.PackageName)
	files := map[string]string{
		filepath.Join(baseDir, domain.PackageName+".go"):   renderDomainFile(domain),
		filepath.Join(baseDir, "service.go"):               renderServiceFile(domain),
		filepath.Join(baseDir, "error.go"):                 renderErrorFile(domain),
		filepath.Join(baseDir, "postgresql", "query.sql"):  renderQuerySQL(domain),
		filepath.Join(baseDir, "postgresql", "storage.go"): renderStorageFile(domain),
	}

	for path, content := range files {
		if err := writeFile(path, content, cfg.Overwrite); err != nil {
			return err
		}
	}

	queryPath := fmt.Sprintf("./domain/%s/postgresql/query.sql", domain.PackageName)
	if err := addSQLCQueryPath(cfg.SQLCYAML, queryPath); err != nil {
		return err
	}

	fmt.Printf("generated domain %q for table %s.%s\n", domain.PackageName, table.Schema, table.Name)
	fmt.Println("next: add queries to", filepath.Join(baseDir, "postgresql", "query.sql"))
	return nil
}

func readModulePath(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("module path not found in %s", path)
}

func findCreateTable(sql, target string) (Table, error) {
	targetSchema, targetTable := splitTableName(target)
	re := regexp.MustCompile(`(?is)\bcreate\s+table\s+(?:if\s+not\s+exists\s+)?([a-zA-Z_][\w]*)(?:\.([a-zA-Z_][\w]*))?\s*\(`)
	matches := re.FindAllStringSubmatchIndex(sql, -1)
	for _, m := range matches {
		first := sql[m[2]:m[3]]
		second := ""
		if m[4] >= 0 {
			second = sql[m[4]:m[5]]
		}

		schema := "public"
		tableName := first
		if second != "" {
			schema = first
			tableName = second
		}

		if !equalName(tableName, targetTable) || (targetSchema != "" && !equalName(schema, targetSchema)) {
			continue
		}

		openParen := m[1] - 1
		closeParen, err := matchingParen(sql, openParen)
		if err != nil {
			return Table{}, err
		}

		columns, err := parseColumns(sql[openParen+1 : closeParen])
		if err != nil {
			return Table{}, err
		}

		return Table{
			Schema:  schema,
			Name:    tableName,
			Columns: columns,
		}, nil
	}

	return Table{}, fmt.Errorf("create table %q not found in schema", target)
}

func splitTableName(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", strings.TrimSpace(name)
}

func equalName(a, b string) bool {
	return strings.EqualFold(strings.Trim(a, `"`), strings.Trim(b, `"`))
}

func matchingParen(s string, open int) (int, error) {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("matching parenthesis not found")
}

func parseColumns(body string) ([]Column, error) {
	body = stripLineComments(body)
	parts := splitTopLevel(body, ',')
	columns := make([]Column, 0, len(parts))
	for _, part := range parts {
		col, ok, err := parseColumn(part)
		if err != nil {
			return nil, err
		}
		if ok {
			columns = append(columns, col)
		}
	}
	return columns, nil
}

func stripLineComments(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}

func splitTopLevel(s string, sep rune) []string {
	var parts []string
	start := 0
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if r == sep && depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + len(string(r))
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}

func parseColumn(def string) (Column, bool, error) {
	def = strings.TrimSpace(def)
	if def == "" {
		return Column{}, false, nil
	}

	fields := strings.Fields(def)
	if len(fields) < 2 {
		return Column{}, false, nil
	}

	first := strings.ToLower(strings.Trim(fields[0], `"`))
	switch first {
	case "primary", "foreign", "unique", "check", "constraint", "exclude":
		return Column{}, false, nil
	}

	name := strings.Trim(fields[0], `"`)
	typeParts := make([]string, 0, len(fields)-1)
	for _, field := range fields[1:] {
		lower := strings.ToLower(field)
		if isColumnConstraint(lower) {
			break
		}
		typeParts = append(typeParts, field)
	}
	if len(typeParts) == 0 {
		return Column{}, false, fmt.Errorf("column %s has no type", name)
	}

	sqlType := strings.Join(typeParts, " ")
	lowerDef := strings.ToLower(def)
	col := Column{
		Name:       name,
		SQLType:    sqlType,
		GoType:     goType(sqlType),
		PrimaryKey: strings.Contains(lowerDef, "primary key"),
		HasDefault: strings.Contains(lowerDef, " default "),
		IsSerial:   isSerial(sqlType),
		NotNull:    strings.Contains(lowerDef, "not null") || strings.Contains(lowerDef, "primary key"),
	}
	col.Insertable = !col.PrimaryKey && !col.HasDefault && !col.IsSerial
	return col, true, nil
}

func isColumnConstraint(s string) bool {
	s = strings.TrimRight(s, ",")
	switch s {
	case "primary", "not", "null", "default", "references", "unique", "check", "constraint", "collate", "generated":
		return true
	default:
		return false
	}
}

func goType(sqlType string) string {
	t := strings.ToLower(sqlType)
	t = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(t, " "))
	switch {
	case strings.Contains(t, "bigserial"), strings.Contains(t, "bigint"):
		return "int64"
	case strings.Contains(t, "serial"), strings.Contains(t, "integer"), strings.Contains(t, " int"), t == "int":
		return "int"
	case strings.Contains(t, "smallint"):
		return "int"
	case strings.Contains(t, "timestamp"), t == "date", strings.HasPrefix(t, "time"):
		return "time.Time"
	case strings.Contains(t, "bool"):
		return "bool"
	case strings.Contains(t, "numeric"), strings.Contains(t, "decimal"), strings.Contains(t, "double"), strings.Contains(t, "real"):
		return "float64"
	case strings.Contains(t, "bytea"):
		return "[]byte"
	default:
		return "string"
	}
}

func isSerial(sqlType string) bool {
	t := strings.ToLower(sqlType)
	return strings.Contains(t, "serial")
}

func buildDomain(table Table, domainName, pkgName, module string) Domain {
	importTime := false
	for _, col := range table.Columns {
		if col.GoType == "time.Time" {
			importTime = true
		}
	}

	return Domain{
		PackageName: pkgName,
		TypeName:    pascal(domainName),
		Table:       table,
		Columns:     table.Columns,
		Module:      module,
		ImportTime:  importTime,
	}
}

func renderDomainFile(d Domain) string {
	var b strings.Builder
	b.WriteString("package " + d.PackageName + "\n\n")
	if d.ImportTime {
		b.WriteString("import \"time\"\n\n")
	}

	b.WriteString("type " + d.TypeName + " struct {\n")
	for _, col := range d.Columns {
		b.WriteString(fmt.Sprintf("\t%-12s %s `json:\"%s\"`\n", domainField(col.Name), col.GoType, col.Name))
	}
	b.WriteString("}\n\n")
	b.WriteString("type Reader interface{}\n\n")
	b.WriteString("type Writer interface{}\n\n")
	b.WriteString("type Repository interface {\n")
	b.WriteString("\tReader\n")
	b.WriteString("\tWriter\n")
	b.WriteString("}\n\n")
	b.WriteString("type UseCase interface{}\n")
	return b.String()
}

func renderServiceFile(d Domain) string {
	return fmt.Sprintf(`package %s

import "%s/internal/logger"

type Service struct {
	repo   Repository
	logger logger.Logger
}

var _ UseCase = (*Service)(nil)

func NewService(r Repository, logger logger.Logger) *Service {
	return &Service{
		repo:   r,
		logger: logger,
	}
}
`, d.PackageName, d.Module)
}

func renderErrorFile(d Domain) string {
	return fmt.Sprintf(`package %s

import "fmt"

func NewDatabaseError(operation string, cause error) error {
	return fmt.Errorf("database operation %%q failed: %%w", operation, cause)
}
`, d.PackageName)
}

func renderQuerySQL(d Domain) string {
	return fmt.Sprintf("-- Write sqlc queries for %s.%s here.\n", d.Table.Schema, d.Table.Name)
}

func renderStorageFile(d Domain) string {
	var b strings.Builder
	b.WriteString("package postgresql\n\n")
	b.WriteString("import (\n")
	b.WriteString(fmt.Sprintf("\t\"%s/domain/%s\"\n", d.Module, d.PackageName))
	b.WriteString(fmt.Sprintf("\t\"%s/internal/database/postgresql\"\n", d.Module))
	b.WriteString(")\n\n")
	b.WriteString(fmt.Sprintf("type %sStorage struct {\n", d.TypeName))
	b.WriteString("\tdb *postgresql.Database\n")
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("var _ %s.Repository = (*%sStorage)(nil)\n\n", d.PackageName, d.TypeName))
	b.WriteString(fmt.Sprintf("func New%s(db *postgresql.Database) *%sStorage {\n", d.TypeName, d.TypeName))
	b.WriteString(fmt.Sprintf("\treturn &%sStorage{\n", d.TypeName))
	b.WriteString("\t\tdb: db,\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String()
}

func writeFile(path, content string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use -overwrite to replace it", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func addSQLCQueryPath(path, queryPath string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(b)
	quoted := fmt.Sprintf("\"%s\"", queryPath)
	if strings.Contains(text, quoted) || strings.Contains(text, queryPath) {
		return nil
	}

	lines := strings.Split(text, "\n")
	queryLine := -1
	queryIndent := ""
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "queries:") {
			queryLine = i
			queryIndent = line[:len(line)-len(strings.TrimLeft(line, " "))]
			break
		}
	}
	if queryLine == -1 {
		return fmt.Errorf("queries section not found in %s", path)
	}

	insertAt := queryLine + 1
	for insertAt < len(lines) {
		trimmed := strings.TrimSpace(lines[insertAt])
		if trimmed == "" {
			insertAt++
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			insertAt++
			continue
		}
		break
	}

	newLine := queryIndent + "    - " + quoted
	lines = append(lines[:insertAt], append([]string{newLine}, lines[insertAt:]...)...)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func domainField(name string) string {
	return pascalWithInitialisms(name, map[string]string{
		"id":   "ID",
		"ip":   "IP",
		"url":  "URL",
		"http": "HTTP",
	})
}

func pascalWithInitialisms(name string, initialisms map[string]string) string {
	parts := splitName(name)
	var b strings.Builder
	for _, part := range parts {
		lower := strings.ToLower(part)
		if replacement, ok := initialisms[lower]; ok {
			b.WriteString(replacement)
			continue
		}
		b.WriteString(upperFirst(lower))
	}
	return b.String()
}

func pascal(name string) string {
	return pascalWithInitialisms(name, map[string]string{
		"id": "ID",
	})
}

func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func splitName(name string) []string {
	fields := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			result = append(result, field)
		}
	}
	return result
}

func packageName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func singular(name string) string {
	name = strings.Trim(name, `"`)
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "ies") && len(name) > 3:
		return name[:len(name)-3] + "y"
	case strings.HasSuffix(lower, "ses") && len(name) > 2:
		return name[:len(name)-2]
	case strings.HasSuffix(lower, "s") && len(name) > 1:
		return name[:len(name)-1]
	default:
		return name
	}
}
