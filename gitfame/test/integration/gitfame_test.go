package integration

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"gitlab.com/slon/shad-go/tools/testtool"
)

const importPath = "gitlab.com/slon/shad-go/gitfame/cmd/gitfame"

var binCache testtool.BinCache

func TestMain(m *testing.M) {
	os.Exit(func() int {
		var teardown testtool.CloseFunc
		binCache, teardown = testtool.NewBinCache()
		defer teardown()

		return m.Run()
	}())
}

func TestGitFame(t *testing.T) {
	binary, err := binCache.GetBinary(importPath)
	require.NoError(t, err)

	bundlesDir := path.Join("./testdata", "bundles")
	testsDir := path.Join("./testdata", "tests")
	testDirs := ListTestDirs(t, testsDir)

	for _, dir := range testDirs {
		tc := ReadTestCase(t, filepath.Join(testsDir, dir))

		t.Run(dir+"/"+tc.Name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "gitfame-")
			require.NoError(t, err)
			defer func() { _ = os.RemoveAll(dir) }()

			args := []string{"--repository", dir}
			args = append(args, tc.Args...)

			Unbundle(t, filepath.Join(bundlesDir, tc.Bundle), dir)

			cmd := exec.Command(binary, args...)
			cmd.Stderr = ioutil.Discard

			output, err := cmd.Output()
			if !tc.Error {
				require.NoError(t, err)
				require.Equal(t, string(tc.Expected), string(output))
			} else {
				require.Error(t, err)
				_, ok := err.(*exec.ExitError)
				require.True(t, ok)
			}
		})
	}
}

func ListTestDirs(t *testing.T, path string) []string {
	t.Helper()

	files, err := ioutil.ReadDir(path)
	require.NoError(t, err)

	var names []string
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		names = append(names, f.Name())
	}

	toInt := func(name string) int {
		i, err := strconv.Atoi(name)
		require.NoError(t, err)
		return i
	}

	sort.Slice(names, func(i, j int) bool {
		return toInt(names[i]) < toInt(names[j])
	})

	return names
}

type TestCase struct {
	*TestDescription
	Expected []byte
}

func ReadTestCase(t *testing.T, path string) *TestCase {
	t.Helper()

	desc := ReadTestDescription(t, path)

	expected, err := ioutil.ReadFile(filepath.Join(path, "expected.out"))
	require.NoError(t, err)

	return &TestCase{TestDescription: desc, Expected: expected}
}

type TestDescription struct {
	Name   string   `yaml:"name"`
	Args   []string `yaml:"args"`
	Bundle string   `yaml:"bundle"`
	Error  bool     `yaml:"error"`
}

func ReadTestDescription(t *testing.T, path string) *TestDescription {
	t.Helper()

	data, err := ioutil.ReadFile(filepath.Join(path, "description.yaml"))
	require.NoError(t, err)

	var desc TestDescription
	require.NoError(t, yaml.Unmarshal(data, &desc))

	return &desc
}

func Unbundle(t *testing.T, src, dst string) {
	t.Helper()

	cmd := exec.Command("git", "clone", src, dst)
	require.NoError(t, cmd.Run())
}
