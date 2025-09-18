package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/analysis"
)

// EnhancedExporter provides enhanced export capabilities
type EnhancedExporter struct {
	outputDir string
}

// NewEnhancedExporter creates a new enhanced exporter
func NewEnhancedExporter(outputDir string) *EnhancedExporter {
	if outputDir == "" {
		outputDir = "./exports"
	}
	return &EnhancedExporter{
		outputDir: outputDir,
	}
}

// ExportHTMLReport exports analysis results as an HTML report
func (ee *EnhancedExporter) ExportHTMLReport(report *analysis.ComprehensiveAnalysisReport, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("cloudrecon-report-%d.html", time.Now().Unix())
	}

	// Ensure output directory exists
	if err := os.MkdirAll(ee.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filepath := filepath.Join(ee.outputDir, filename)

	// Generate HTML content
	htmlContent, err := ee.generateHTMLReport(report)
	if err != nil {
		return fmt.Errorf("failed to generate HTML content: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, []byte(htmlContent), 0600); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	fmt.Printf(" HTML report exported to: %s\n", filepath)
	return nil
}

// ExportPDFReport exports analysis results as a PDF report
func (ee *EnhancedExporter) ExportPDFReport(report *analysis.ComprehensiveAnalysisReport, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("cloudrecon-report-%d.pdf", time.Now().Unix())
	}

	// Ensure output directory exists
	if err := os.MkdirAll(ee.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filepath := filepath.Join(ee.outputDir, filename)

	// For now, we'll generate an HTML file that can be converted to PDF
	// In a real implementation, you would use a PDF library like gofpdf or unidoc
	htmlContent, err := ee.generatePDFHTMLReport(report)
	if err != nil {
		return fmt.Errorf("failed to generate PDF HTML content: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, []byte(htmlContent), 0600); err != nil {
		return fmt.Errorf("failed to write PDF file: %w", err)
	}

	fmt.Printf(" PDF report exported to: %s (HTML format, ready for PDF conversion)\n", filepath)
	return nil
}

// generateHTMLReport generates the HTML content for the report
func (ee *EnhancedExporter) generateHTMLReport(report *analysis.ComprehensiveAnalysisReport) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CloudRecon Analysis Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .header p {
            margin: 10px 0 0 0;
            opacity: 0.9;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            text-align: center;
        }
        .summary-card h3 {
            margin: 0 0 10px 0;
            color: #666;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .summary-card .value {
            font-size: 2em;
            font-weight: bold;
            color: #333;
        }
        .section {
            background: white;
            padding: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        .section h2 {
            margin: 0 0 20px 0;
            color: #333;
            border-bottom: 2px solid #667eea;
            padding-bottom: 10px;
        }
        .finding {
            padding: 15px;
            margin: 10px 0;
            border-left: 4px solid #e74c3c;
            background: #f8f9fa;
            border-radius: 0 5px 5px 0;
        }
        .finding.critical { border-left-color: #e74c3c; }
        .finding.high { border-left-color: #f39c12; }
        .finding.medium { border-left-color: #f1c40f; }
        .finding.low { border-left-color: #27ae60; }
        .finding h4 {
            margin: 0 0 5px 0;
            color: #333;
        }
        .finding .severity {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            font-weight: bold;
            text-transform: uppercase;
            margin-right: 10px;
        }
        .finding.critical .severity { background: #e74c3c; color: white; }
        .finding.high .severity { background: #f39c12; color: white; }
        .finding.medium .severity { background: #f1c40f; color: #333; }
        .finding.low .severity { background: #27ae60; color: white; }
        .optimization {
            padding: 15px;
            margin: 10px 0;
            border-left: 4px solid #27ae60;
            background: #f8f9fa;
            border-radius: 0 5px 5px 0;
        }
        .optimization h4 {
            margin: 0 0 5px 0;
            color: #333;
        }
        .optimization .savings {
            color: #27ae60;
            font-weight: bold;
        }
        .dependency {
            padding: 10px;
            margin: 5px 0;
            background: #f8f9fa;
            border-radius: 5px;
            font-family: monospace;
            font-size: 0.9em;
        }
        .footer {
            text-align: center;
            color: #666;
            margin-top: 50px;
            padding: 20px;
            border-top: 1px solid #ddd;
        }
        .chart {
            width: 100%;
            height: 300px;
            background: #f8f9fa;
            border-radius: 5px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #666;
            font-style: italic;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1> CloudRecon Analysis Report</h1>
        <p>Generated on {{.GeneratedAt.Format "January 2, 2006 at 3:04 PM MST"}}</p>
    </div>

    <div class="summary">
        <div class="summary-card">
            <h3>Total Resources</h3>
            <div class="value">{{.Summary.TotalResources}}</div>
        </div>
        <div class="summary-card">
            <h3>Dependencies</h3>
            <div class="value">{{.Summary.TotalDependencies}}</div>
        </div>
        <div class="summary-card">
            <h3>Security Findings</h3>
            <div class="value">{{.Summary.SecurityFindings}}</div>
        </div>
        <div class="summary-card">
            <h3>Monthly Cost</h3>
            <div class="value">${{printf "%.2f" .Summary.TotalMonthlyCost}}</div>
        </div>
        <div class="summary-card">
            <h3>Compliance Score</h3>
            <div class="value">{{printf "%.1f" .Summary.ComplianceScore}}%</div>
        </div>
        <div class="summary-card">
            <h3>Risk Score</h3>
            <div class="value">{{printf "%.1f" .Summary.RiskScore}}</div>
        </div>
    </div>

    {{if .Security}}
    <div class="section">
        <h2> Security Analysis</h2>
        {{if .Security.Findings}}
            {{range .Security.Findings}}
            <div class="finding {{.Severity}}">
                <h4>
                    <span class="severity">{{.Severity}}</span>
                    {{.Title}}
                </h4>
                <p>{{.Description}}</p>
                <p><strong>Recommendation:</strong> {{.Recommendation}}</p>
            </div>
            {{end}}
        {{else}}
            <p> No security findings detected.</p>
        {{end}}
    </div>
    {{end}}

    {{if .Cost}}
    <div class="section">
        <h2> Cost Analysis</h2>
        {{if .Cost.Optimizations}}
            {{range .Cost.Optimizations}}
            <div class="optimization">
                <h4>{{.Title}}</h4>
                <p>{{.Description}}</p>
                <p><strong>Potential Savings:</strong> <span class="savings">${{printf "%.2f" .PotentialSavings}}/month ({{printf "%.1f" .SavingsPercent}}%)</span></p>
            </div>
            {{end}}
        {{else}}
            <p> No cost optimizations identified.</p>
        {{end}}
    </div>
    {{end}}

    {{if .Dependencies}}
    <div class="section">
        <h2> Dependency Analysis</h2>
        <p><strong>Total Dependencies:</strong> {{.Dependencies.Stats.TotalDependencies}}</p>
        <p><strong>Cycles:</strong> {{.Dependencies.Stats.Cycles}}</p>
        <p><strong>Islands:</strong> {{.Dependencies.Stats.Islands}}</p>
        <p><strong>Max Depth:</strong> {{.Dependencies.Stats.MaxDepth}}</p>
    </div>
    {{end}}

    <div class="footer">
        <p>Report generated by CloudRecon - Multi-Cloud Discovery Tool</p>
    </div>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, report); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generatePDFHTMLReport generates HTML content optimized for PDF conversion
func (ee *EnhancedExporter) generatePDFHTMLReport(report *analysis.ComprehensiveAnalysisReport) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CloudRecon Analysis Report</title>
    <style>
        @page {
            margin: 1in;
            size: A4;
        }
        body {
            font-family: Arial, sans-serif;
            line-height: 1.4;
            color: #333;
            font-size: 12px;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 2px solid #333;
        }
        .header h1 {
            margin: 0;
            font-size: 24px;
            color: #333;
        }
        .header p {
            margin: 5px 0 0 0;
            color: #666;
        }
        .summary {
            display: table;
            width: 100%;
            margin-bottom: 30px;
        }
        .summary-row {
            display: table-row;
        }
        .summary-cell {
            display: table-cell;
            width: 16.66%;
            padding: 10px;
            text-align: center;
            border: 1px solid #ddd;
        }
        .summary-cell h3 {
            margin: 0 0 5px 0;
            font-size: 10px;
            color: #666;
            text-transform: uppercase;
        }
        .summary-cell .value {
            font-size: 18px;
            font-weight: bold;
        }
        .section {
            margin-bottom: 30px;
            page-break-inside: avoid;
        }
        .section h2 {
            margin: 0 0 15px 0;
            font-size: 16px;
            color: #333;
            border-bottom: 1px solid #333;
            padding-bottom: 5px;
        }
        .finding {
            margin: 10px 0;
            padding: 10px;
            border-left: 3px solid #e74c3c;
            background: #f8f9fa;
        }
        .finding h4 {
            margin: 0 0 5px 0;
            font-size: 12px;
        }
        .finding .severity {
            font-weight: bold;
            text-transform: uppercase;
        }
        .optimization {
            margin: 10px 0;
            padding: 10px;
            border-left: 3px solid #27ae60;
            background: #f8f9fa;
        }
        .optimization h4 {
            margin: 0 0 5px 0;
            font-size: 12px;
        }
        .footer {
            text-align: center;
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            font-size: 10px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>CloudRecon Analysis Report</h1>
        <p>Generated on {{.GeneratedAt.Format "January 2, 2006 at 3:04 PM MST"}}</p>
    </div>

    <div class="summary">
        <div class="summary-row">
            <div class="summary-cell">
                <h3>Resources</h3>
                <div class="value">{{.Summary.TotalResources}}</div>
            </div>
            <div class="summary-cell">
                <h3>Dependencies</h3>
                <div class="value">{{.Summary.TotalDependencies}}</div>
            </div>
            <div class="summary-cell">
                <h3>Security Findings</h3>
                <div class="value">{{.Summary.SecurityFindings}}</div>
            </div>
            <div class="summary-cell">
                <h3>Monthly Cost</h3>
                <div class="value">${{printf "%.2f" .Summary.TotalMonthlyCost}}</div>
            </div>
            <div class="summary-cell">
                <h3>Compliance</h3>
                <div class="value">{{printf "%.1f" .Summary.ComplianceScore}}%</div>
            </div>
            <div class="summary-cell">
                <h3>Risk Score</h3>
                <div class="value">{{printf "%.1f" .Summary.RiskScore}}</div>
            </div>
        </div>
    </div>

    {{if .Security}}
    <div class="section">
        <h2>Security Analysis</h2>
        {{if .Security.Findings}}
            {{range .Security.Findings}}
            <div class="finding">
                <h4><span class="severity">{{.Severity}}</span> {{.Title}}</h4>
                <p>{{.Description}}</p>
                <p><strong>Recommendation:</strong> {{.Recommendation}}</p>
            </div>
            {{end}}
        {{else}}
            <p>No security findings detected.</p>
        {{end}}
    </div>
    {{end}}

    {{if .Cost}}
    <div class="section">
        <h2>Cost Analysis</h2>
        {{if .Cost.Optimizations}}
            {{range .Cost.Optimizations}}
            <div class="optimization">
                <h4>{{.Title}}</h4>
                <p>{{.Description}}</p>
                <p><strong>Potential Savings:</strong> ${{printf "%.2f" .PotentialSavings}}/month ({{printf "%.1f" .SavingsPercent}}%)</p>
            </div>
            {{end}}
        {{else}}
            <p>No cost optimizations identified.</p>
        {{end}}
    </div>
    {{end}}

    {{if .Dependencies}}
    <div class="section">
        <h2>Dependency Analysis</h2>
        <p><strong>Total Dependencies:</strong> {{.Dependencies.Stats.TotalDependencies}}</p>
        <p><strong>Cycles:</strong> {{.Dependencies.Stats.Cycles}}</p>
        <p><strong>Islands:</strong> {{.Dependencies.Stats.Islands}}</p>
        <p><strong>Max Depth:</strong> {{.Dependencies.Stats.MaxDepth}}</p>
    </div>
    {{end}}

    <div class="footer">
        <p>Report generated by CloudRecon - Multi-Cloud Discovery Tool</p>
    </div>
</body>
</html>`

	t, err := template.New("pdf-report").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, report); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ExportJSONReport exports analysis results as a JSON report
func (ee *EnhancedExporter) ExportJSONReport(report *analysis.ComprehensiveAnalysisReport, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("cloudrecon-report-%d.json", time.Now().Unix())
	}

	// Ensure output directory exists
	if err := os.MkdirAll(ee.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filepath := filepath.Join(ee.outputDir, filename)

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	fmt.Printf(" JSON report exported to: %s\n", filepath)
	return nil
}

// ExportCustomFormat exports analysis results in a custom format
func (ee *EnhancedExporter) ExportCustomFormat(report *analysis.ComprehensiveAnalysisReport, format string, filename string) error {
	switch format {
	case "html":
		return ee.ExportHTMLReport(report, filename)
	case "pdf":
		return ee.ExportPDFReport(report, filename)
	case "json":
		return ee.ExportJSONReport(report, filename)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportAllFormats exports analysis results in all available formats
func (ee *EnhancedExporter) ExportAllFormats(report *analysis.ComprehensiveAnalysisReport, baseFilename string) error {
	if baseFilename == "" {
		baseFilename = fmt.Sprintf("cloudrecon-report-%d", time.Now().Unix())
	}

	formats := []string{"html", "pdf"}

	for _, format := range formats {
		filename := fmt.Sprintf("%s.%s", baseFilename, format)
		if err := ee.ExportCustomFormat(report, format, filename); err != nil {
			return fmt.Errorf("failed to export %s format: %w", format, err)
		}
	}

	return nil
}
