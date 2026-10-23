package runner_test

import (
	"path/filepath"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/runner"
)

func TestVendorStatus(t *testing.T) {
	t.Parallel()

	root, source := vendorFixture(t)
	writeFile(t, filepath.Join(source, "go.yml"), goSkill)
	writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

	status := func() runner.VendorStatus {
		result, err := runner.NewVendorer(source, root).Plan()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Used))
		return result.Used[0].Status
	}

	assert.Equal(t, runner.VendorNew, status())

	_, err := runner.NewVendorer(source, root).Run()
	assert.NoError(t, err)
	assert.Equal(t, runner.VendorCurrent, status())

	writeFile(t, filepath.Join(root, ".atkins", "skills", "go.yml"), goSkill+"# local edit\n")
	assert.Equal(t, runner.VendorChanged, status())
}

func TestVendorSkill(t *testing.T) {
	t.Parallel()

	root, source := vendorFixture(t)
	writeFile(t, filepath.Join(source, "go.yml"), goSkill)
	writeFile(t, filepath.Join(source, "go.md"), "# Go\n")
	writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

	result, err := runner.NewVendorer(source, root).Plan()
	assert.NoError(t, err)

	var skill *runner.VendorSkill = result.Used[0]
	assert.Equal(t, "go", skill.ID)
	assert.Equal(t, filepath.Join(source, "go.yml"), skill.Path)
	assert.Equal(t, filepath.Join(source, "go.md"), skill.DocPath)
	assert.Equal(t, "go.mod", skill.Reason)
	assert.NotNil(t, skill.Pipeline)
	assert.True(t, skill.Added > 0, "a new skill adds its lines")
}
