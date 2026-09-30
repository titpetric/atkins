# Package ./internal/assert

```go
import (
	"github.com/titpetric/atkins/internal/assert"
)
```

Package assert is a minimalistic drop-in replacement for testify/assert.
It is based on fortio.org/assert and is internal to this module.

## Types

<details>
<summary><code>type TestSuite</code></summary>

```go
// TestSuite to be used as base struct for test suites.
// Replaces https://pkg.go.dev/github.com/stretchr/testify@v1.8.0/suite.
type TestSuite struct {
	t *testing.T
}
```

</details>

## Vars

<details>
<summary><code>var AnError</code></summary>

```go
// AnError is a sentinel error for tests that need one and do not care
// which. It replaces testify's assert.AnError.
var AnError = errors.New("assert.AnError general error for testing")
```

</details>

## Function symbols

- `func Assert (t testing.TB, cond bool, msg string, rest ...any)`
- `func CheckEquals (t testing.TB, actual,expected any, msg ...any)`
- `func Contains (t testing.TB, haystack,needle any, msg ...any)`
- `func DirExists (t testing.TB, path string, msg ...any)`
- `func ElementsMatch (t testing.TB, listA,listB any, msg ...any)`
- `func Empty (t testing.TB, value any, msg ...any)`
- `func Equal (t testing.TB, a,b any, msg ...any)`
- `func EqualValues (t testing.TB, a,b any, msg ...any)`
- `func Error (t testing.TB, err error, msg ...any)`
- `func ErrorAs (t testing.TB, err error, target any, msg ...any)`
- `func ErrorContains (t testing.TB, err error, want string, msg ...any)`
- `func ErrorIs (t testing.TB, err,target error, msg ...any)`
- `func Errorf (t testing.TB, format string, rest ...any)`
- `func Eventually (t testing.TB, condition func() bool, waitFor,tick time.Duration, msg ...any)`
- `func Fail (t testing.TB, msg string, args ...any)`
- `func False (t testing.TB, b bool, msg ...any)`
- `func FileExists (t testing.TB, path string, msg ...any)`
- `func Greater (t testing.TB, actual,expected any, msg ...any)`
- `func GreaterOrEqual (t testing.TB, actual,expected any, msg ...any)`
- `func IsIncreasing (t testing.TB, value any, msg ...any)`
- `func JSONEq (t testing.TB, expected,actual string, msg ...any)`
- `func Len (t testing.TB, value any, expected int, msg ...any)`
- `func Less (t testing.TB, actual,expected any, msg ...any)`
- `func LessOrEqual (t testing.TB, actual,expected any, msg ...any)`
- `func Nil (t testing.TB, value any, msg ...any)`
- `func NoDirExists (t testing.TB, path string, msg ...any)`
- `func NoError (t testing.TB, err error, msg ...any)`
- `func NoFileExists (t testing.TB, path string, msg ...any)`
- `func NotContains (t testing.TB, haystack,needle any, msg ...any)`
- `func NotEmpty (t testing.TB, value any, msg ...any)`
- `func NotEqual (t testing.TB, a,b any, msg ...any)`
- `func NotNil (t testing.TB, value any, msg ...any)`
- `func NotZero (t testing.TB, value any, msg ...any)`
- `func ObjectsAreEqualValues (a,b any) bool`
- `func Positive (t testing.TB, value any, msg ...any)`
- `func Run (t *testing.T, suite hasT)`
- `func True (t testing.TB, b bool, msg ...any)`
- `func Zero (t testing.TB, value any, msg ...any)`
- `func (*TestSuite) SetT (t *testing.T)`
- `func (*TestSuite) T () *testing.T`

### Assert

Assert is similar to True() under a different name and earlier
fortio/stats test implementation.

```go
func Assert(t testing.TB, cond bool, msg string, rest ...any)
```

### CheckEquals

CheckEquals checks if actual == expect and fails the test and logs
failure (including filename:linenum if they are not equal).

```go
func CheckEquals(t testing.TB, actual, expected any, msg ...any)
```

### Contains

Contains checks that needle is in haystack.

```go
func Contains(t testing.TB, haystack, needle any, msg ...any)
```

### DirExists

DirExists checks that path exists and is a directory.

```go
func DirExists(t testing.TB, path string, msg ...any)
```

### ElementsMatch

ElementsMatch checks that two arrays or slices hold the same elements
with the same multiplicity, in any order.

```go
func ElementsMatch(t testing.TB, listA, listB any, msg ...any)
```

### Empty

Empty checks that value is nil, zero, or has length zero.

```go
func Empty(t testing.TB, value any, msg ...any)
```

### Equal

Equal also checks for a equal b.

```go
func Equal(t testing.TB, a, b any, msg ...any)
```

### EqualValues

EqualValues checks for a equal b.

```go
func EqualValues(t testing.TB, a, b any, msg ...any)
```

### Error

Error checks/expects an error.

```go
func Error(t testing.TB, err error, msg ...any)
```

### ErrorAs

ErrorAs checks that err, or an error it wraps, is assignable to
target, which must be a non-nil pointer.

```go
func ErrorAs(t testing.TB, err error, target any, msg ...any)
```

### ErrorContains

ErrorContains checks that err is non-nil and its message contains want.

```go
func ErrorContains(t testing.TB, err error, want string, msg ...any)
```

### ErrorIs

ErrorIs checks that err matches target.

```go
func ErrorIs(t testing.TB, err, target error, msg ...any)
```

### Errorf

Errorf is a local variant to get the right line numbers.

```go
func Errorf(t testing.TB, format string, rest ...any)
```

### Eventually

Eventually polls condition every tick until it returns true or
waitFor elapses.

```go
func Eventually(t testing.TB, condition func() bool, waitFor, tick time.Duration, msg ...any)
```

### Fail

Fail fails the test.

```go
func Fail(t testing.TB, msg string, args ...any)
```

### False

False checks bool is false.

```go
func False(t testing.TB, b bool, msg ...any)
```

### FileExists

FileExists checks that path exists and is a regular file.

```go
func FileExists(t testing.TB, path string, msg ...any)
```

### Greater

Greater checks that actual is greater than expected.

```go
func Greater(t testing.TB, actual, expected any, msg ...any)
```

### GreaterOrEqual

GreaterOrEqual checks that actual is greater than or equal to expected.

```go
func GreaterOrEqual(t testing.TB, actual, expected any, msg ...any)
```

### IsIncreasing

IsIncreasing checks that every item is greater than the item before it.

```go
func IsIncreasing(t testing.TB, value any, msg ...any)
```

### JSONEq

JSONEq checks that two JSON documents carry the same value, ignoring
key order and whitespace.

```go
func JSONEq(t testing.TB, expected, actual string, msg ...any)
```

### Len

Len checks the length of a string, array, channel, map, or slice.

```go
func Len(t testing.TB, value any, expected int, msg ...any)
```

### Less

Less checks that actual is less than expected.

```go
func Less(t testing.TB, actual, expected any, msg ...any)
```

### LessOrEqual

LessOrEqual checks that actual is less than or equal to expected.

```go
func LessOrEqual(t testing.TB, actual, expected any, msg ...any)
```

### Nil

Nil checks that value is nil, including a typed nil.

```go
func Nil(t testing.TB, value any, msg ...any)
```

### NoDirExists

NoDirExists checks that path is not a directory. A regular file at
path passes, matching testify.

```go
func NoDirExists(t testing.TB, path string, msg ...any)
```

### NoError

NoError checks for no errors (nil).

```go
func NoError(t testing.TB, err error, msg ...any)
```

### NoFileExists

NoFileExists checks that path is not a regular file. A directory at
path passes, matching testify.

```go
func NoFileExists(t testing.TB, path string, msg ...any)
```

### NotContains

NotContains checks that needle is not in haystack.

```go
func NotContains(t testing.TB, haystack, needle any, msg ...any)
```

### NotEmpty

NotEmpty checks that value is neither nil nor empty.

```go
func NotEmpty(t testing.TB, value any, msg ...any)
```

### NotEqual

NotEqual checks for a not equal b.

```go
func NotEqual(t testing.TB, a, b any, msg ...any)
```

### NotNil

NotNil checks that value is not nil, including a typed nil.

```go
func NotNil(t testing.TB, value any, msg ...any)
```

### NotZero

NotZero checks that value is not the zero value for its type.

```go
func NotZero(t testing.TB, value any, msg ...any)
```

### ObjectsAreEqualValues

ObjectsAreEqualValues returns true if a == b (through refection).

```go
func ObjectsAreEqualValues(a, b any) bool
```

### Positive

Positive checks that value is a number greater than zero.

```go
func Positive(t testing.TB, value any, msg ...any)
```

### Run

Run runs the test suite with SetupTest first and TearDownTest after.
Replaces https://pkg.go.dev/github.com/stretchr/testify/suite#Run.

```go
func Run(t *testing.T, suite hasT)
```

### True

True checks bool is true.

```go
func True(t testing.TB, b bool, msg ...any)
```

### Zero

Zero checks that value is the zero value for its type.

```go
func Zero(t testing.TB, value any, msg ...any)
```

### SetT

SetT sets the testing.T in the suite object.

```go
func (*TestSuite) SetT(t *testing.T)
```

### T

T returns the current testing.T.

```go
func (*TestSuite) T() *testing.T
```
