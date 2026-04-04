package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/itchyny/gojq"
	"github.com/spf13/pflag"
)

type apiOutputOptions struct {
	jqExpr   string
	jsonExpr string
	template string
}

func bindAPIOutputFlags(flags *pflag.FlagSet, options *apiOutputOptions) {
	flags.StringVar(&options.jqExpr, "jq", "", "Apply a jq expression to the JSON response.")
	flags.StringVar(&options.jsonExpr, "json", "", "Select a comma-separated list of JSON fields from the response.")
	flags.StringVar(&options.template, "template", "", "Render the JSON response with a Go text/template.")
}

func (options apiOutputOptions) validate() error {
	count := 0
	for _, candidate := range []string{options.jqExpr, options.jsonExpr, options.template} {
		if strings.TrimSpace(candidate) != "" {
			count++
		}
	}
	if count > 1 {
		return fmt.Errorf("only one of --jq, --json, or --template can be set")
	}
	return nil
}

func writeAPIOutput(out io.Writer, body []byte, options apiOutputOptions) error {
	if err := options.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(options.jqExpr) == "" && strings.TrimSpace(options.jsonExpr) == "" && strings.TrimSpace(options.template) == "" {
		return writePrettyJSON(out, body)
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}

	switch {
	case strings.TrimSpace(options.jqExpr) != "":
		return writeJQOutput(out, payload, options.jqExpr)
	case strings.TrimSpace(options.jsonExpr) != "":
		fields := splitCSV(options.jsonExpr)
		trimmed := trimNonEmpty(fields)
		if len(trimmed) == 0 {
			return fmt.Errorf("--json requires at least one field")
		}
		selected := selectJSONFields(payload, trimmed)
		return writeJSONValue(out, selected)
	case strings.TrimSpace(options.template) != "":
		return writeTemplateOutput(out, payload, options.template)
	default:
		return writePrettyJSON(out, body)
	}
}

func writeJQOutput(out io.Writer, payload any, expression string) error {
	query, err := gojq.Parse(expression)
	if err != nil {
		return fmt.Errorf("parse jq expression: %w", err)
	}
	iterator := query.Run(payload)

	wrote := false
	for {
		value, ok := iterator.Next()
		if !ok {
			break
		}
		if iterErr, ok := value.(error); ok {
			return fmt.Errorf("evaluate jq expression: %w", iterErr)
		}
		wrote = true
		switch typed := value.(type) {
		case string:
			if _, err := fmt.Fprintln(out, typed); err != nil {
				return err
			}
		case nil:
			if _, err := fmt.Fprintln(out, "null"); err != nil {
				return err
			}
		default:
			encoded, marshalErr := json.Marshal(typed)
			if marshalErr != nil {
				return fmt.Errorf("marshal jq result: %w", marshalErr)
			}
			if _, err := fmt.Fprintln(out, string(encoded)); err != nil {
				return err
			}
		}
	}
	if wrote {
		return nil
	}
	_, err = fmt.Fprintln(out, "null")
	return err
}

func writeTemplateOutput(out io.Writer, payload any, templateText string) error {
	tmpl, err := template.New("output").Option("missingkey=zero").Parse(templateText)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, payload); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if buffer.Len() == 0 || buffer.Bytes()[buffer.Len()-1] != '\n' {
		buffer.WriteByte('\n')
	}
	_, err = out.Write(buffer.Bytes())
	return err
}

func writeJSONValue(out io.Writer, payload any) error {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON output: %w", err)
	}
	encoded = append(encoded, '\n')
	_, err = out.Write(encoded)
	return err
}

func selectJSONFields(payload any, fields []string) any {
	switch typed := payload.(type) {
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, selectJSONFields(item, fields))
		}
		return items
	case map[string]any:
		selected := make(map[string]any, len(fields))
		for _, field := range fields {
			value, ok := lookupJSONField(typed, strings.Split(field, "."))
			if ok {
				selected[field] = value
			}
		}
		return selected
	default:
		return payload
	}
}

func lookupJSONField(payload any, path []string) (any, bool) {
	if len(path) == 0 {
		return payload, true
	}

	object, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	next, ok := object[path[0]]
	if !ok {
		return nil, false
	}
	return lookupJSONField(next, path[1:])
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		items = append(items, strings.TrimSpace(part))
	}
	return items
}

func trimNonEmpty(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}
