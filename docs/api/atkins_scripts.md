# Package ./scripts

```go
import (
	"github.com/titpetric/atkins/scripts"
}
```

## Types

```go
// CoverageBlock represents a covered line in coverage data
type CoverageBlock struct {
	StartLine, EndLine	int
	StartCol, EndCol	int
	Stmts, Count		int
	Filename		string
	Covered			bool
	Count2			int	// Track if covered by multiple tests
}
```

```go
// CoverageProfile stores all coverage data keyed by file and line range
type CoverageProfile struct {
	Mode	string
	Blocks	map[string]map[string]*CoverageBlock
}
```

