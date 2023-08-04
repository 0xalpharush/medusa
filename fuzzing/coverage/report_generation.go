package coverage

import (
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/utils"
)

var (
	//go:embed report_template.gohtml
	htmlReportTemplate []byte
)
var (
	//go:embed report_template.gojson
	jsonReportTemplate []byte
)

// GenerateReport takes a set of CoverageMaps and compilations, and produces a coverage report using them, detailing
// all source mapped ranges of the source files which were covered or not.
// Returns an error if one occurred.
func GenerateReport(compilations []types.Compilation, coverageMaps *CoverageMaps, reportPath string) error {
	// Perform source analysis.
	sourceAnalysis, err := AnalyzeSourceCoverage(compilations, coverageMaps)
	if err != nil {
		return err
	}

	// Finally, export the report data we analyzed.
	if reportPath != "" {
		err = exportCoverageReport(sourceAnalysis, reportPath)
		err = exportCoverageReportJSON(sourceAnalysis, reportPath)
	}
	return err
}

// exportCoverageReportJSON takes a previously performed source analysis and generates a JSON coverage report from it.
// Returns an error if one occurs.
func exportCoverageReportJSON(sourceAnalysis *SourceAnalysis, outputPath string) error {
	functionMap := template.FuncMap{
		"add": func(x int, y int) int {
			return x + y
		},
		"sub": func(x int, y int) int {
			return x - y
		},
		"relativePath": func(path string) string {
			// Obtain a path relative to our current working directory.
			// If we encounter an error, return the original path.
			cwd, err := os.Getwd()
			if err != nil {
				return path
			}
			relativePath, err := filepath.Rel(cwd, path)
			if err != nil {
				return path
			}

			return relativePath
		},
		"lastActiveIndex": func(sourceFileAnalysis *SourceFileAnalysis) int {
			// Determine the last active line index and return it
			lastIndex := 0
			for lineIndex, line := range sourceFileAnalysis.Lines {
				if line.IsActive {
					lastIndex = lineIndex
				}
			}
			return lastIndex
		},
	}

	// Parse our JSON template
	tmpl, err := template.New("coverage_report.json").Funcs(functionMap).Parse(string(jsonReportTemplate))

	// Add the report file to the path
	outputPath = filepath.Join(outputPath, "coverage_report.json")

	fmt.Println("Report path:", outputPath)

	// If the parent directory doesn't exist, create it.
	parentDirectory := filepath.Dir(outputPath)
	err = utils.MakeDirectory(parentDirectory)
	if err != nil {
		return err
	}

	// Create our report file
	file, err := os.Create(outputPath)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("could not export report, failed to open file for writing: %v", err)
	}

	// Execute the template and write it back to file.
	err = tmpl.Execute(file, sourceAnalysis)
	fileCloseErr := file.Close()
	if err == nil {
		err = fileCloseErr
	}

	return err
}

// exportCoverageReport takes a previously performed source analysis and generates an HTML coverage report from it.
// Returns an error if one occurs.
func exportCoverageReport(sourceAnalysis *SourceAnalysis, outputPath string) error {
	// Define mappings onto some useful variables/functions.
	functionMap := template.FuncMap{
		"timeNow": time.Now,
		"add": func(x int, y int) int {
			return x + y
		},
		"relativePath": func(path string) string {
			// Obtain a path relative to our current working directory.
			// If we encounter an error, return the original path.
			cwd, err := os.Getwd()
			if err != nil {
				return path
			}
			relativePath, err := filepath.Rel(cwd, path)
			if err != nil {
				return path
			}

			return relativePath
		},
		"percentageStr": func(x int, y int, decimals int) string {
			// Determine our precision string
			formatStr := "%." + strconv.Itoa(decimals) + "f"

			// If no lines are active and none are covered, show 100% coverage
			if x == 0 && y == 0 {
				return fmt.Sprintf(formatStr, float64(100))
			}
			return fmt.Sprintf(formatStr, (float64(x)/float64(y))*100)
		},
		"percentageInt": func(x int, y int) int {
			if y == 0 {
				return 100
			}
			return int(math.Round(float64(x) / float64(y) * 100))
		},
	}

	// Parse our HTML template
	tmpl, err := template.New("coverage_report.html").Funcs(functionMap).Parse(string(htmlReportTemplate))
	if err != nil {
		return fmt.Errorf("could not export report, failed to parse report template: %v", err)
	}

	// Add the report file to the path
	outputPath = filepath.Join(outputPath, "coverage_report.html")

	// If the parent directory doesn't exist, create it.
	parentDirectory := filepath.Dir(outputPath)
	err = utils.MakeDirectory(parentDirectory)
	if err != nil {
		return err
	}

	// Create our report file
	file, err := os.Create(outputPath)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("could not export report, failed to open file for writing: %v", err)
	}

	// Execute the template and write it back to file.
	err = tmpl.Execute(file, sourceAnalysis)
	fileCloseErr := file.Close()
	if err == nil {
		err = fileCloseErr
	}
	return err
}
