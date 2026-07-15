package analyzer_test

import (
	"testing"

	"github.com/cago-frame/cagolint/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.New(analyzer.Settings{}),
		"example.com/project/internal/api/badapi",
		"example.com/project/internal/controller/bad_ctr",
		"example.com/project/internal/repository/user_repo",
	)
}

func TestSuggestedFixes(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, analyzer.New(analyzer.Settings{}),
		"example.com/project/internal/api/fixapi",
		"example.com/project/internal/repository/fix_repo",
	)
}
