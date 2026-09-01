package cli

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/codcod/jerry/internal/rules"
)

// FindingsEnvelopeSchema versions the machine-readable findings output.
// Consumers pin it, so changes are additive only.
const FindingsEnvelopeSchema = "jerry.findings/1"

// findingsEnvelope is the --format json shape.
type findingsEnvelope struct {
	Schema   string         `json:"schema"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
	Findings rules.Findings `json:"findings"`
}

// outputFormats are the values --format accepts. sarif exists because it is
// what puts a finding inline on the diff in GitLab and GitHub, which is the
// difference between "CI failed, go read the log" and "line 4 is wrong".
var outputFormats = []string{"text", "json", "sarif", "junit"}

func validateFormat(value string) error {
	for _, candidate := range outputFormats {
		if candidate == value {
			return nil
		}
	}
	return fmt.Errorf("unknown format %q (want one of: %s)", value, strings.Join(outputFormats, ", "))
}

func renderFindings(w io.Writer, findings rules.Findings, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(findingsEnvelope{
			Schema:   FindingsEnvelopeSchema,
			Errors:   findings.Errors(),
			Warnings: findings.Warnings(),
			Findings: findings,
		})
	case "sarif":
		return renderSARIF(w, findings)
	case "junit":
		return renderJUnit(w, findings)
	default:
		return renderText(w, findings)
	}
}

func renderText(w io.Writer, findings rules.Findings) error {
	for _, finding := range findings {
		if _, err := fmt.Fprintln(w, finding.String()); err != nil {
			return err
		}
	}
	return nil
}

// --- SARIF ----------------------------------------------------------------

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID string `json:"id"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func renderSARIF(w io.Writer, findings rules.Findings) error {
	seen := map[string]bool{}
	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool:    sarifTool{Driver: sarifDriver{Name: "jerry", InformationURI: "https://github.com/codcod/jerry"}},
			Results: []sarifResult{},
		}},
	}
	for _, finding := range findings {
		if !seen[finding.Rule] {
			seen[finding.Rule] = true
			log.Runs[0].Tool.Driver.Rules = append(log.Runs[0].Tool.Driver.Rules, sarifRule{ID: finding.Rule})
		}
		level := "error"
		if finding.Severity == rules.SeverityWarning {
			level = "warning"
		}
		log.Runs[0].Results = append(log.Runs[0].Results, sarifResult{
			RuleID:  finding.Rule,
			Level:   level,
			Message: sarifMessage{Text: finding.Message},
			Locations: []sarifLocation{{PhysicalLocation: sarifPhysical{
				ArtifactLocation: sarifArtifact{URI: finding.Path},
				Region:           sarifRegion{StartLine: max(finding.Line, 1)},
			}}},
		})
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(log)
}

// --- JUnit ----------------------------------------------------------------

type junitSuites struct {
	XMLName xml.Name     `xml:"testsuites"`
	Suites  []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Cases    []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
}

func renderJUnit(w io.Writer, findings rules.Findings) error {
	suite := junitSuite{Name: "jerry validate"}
	for _, finding := range findings {
		testCase := junitCase{
			Name:      fmt.Sprintf("%s:%d %s", finding.Path, finding.Line, finding.Rule),
			ClassName: finding.Path,
		}
		if finding.Severity == rules.SeverityError {
			suite.Failures++
			testCase.Failure = &junitFailure{Message: finding.Message, Type: finding.Rule}
		}
		suite.Cases = append(suite.Cases, testCase)
	}
	// A suite with no cases still needs one passing case, or report readers
	// show "no tests ran" rather than "validation passed".
	if len(suite.Cases) == 0 {
		suite.Cases = append(suite.Cases, junitCase{Name: "no findings", ClassName: "jerry"})
	}
	suite.Tests = len(suite.Cases)

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(junitSuites{Suites: []junitSuite{suite}}); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
