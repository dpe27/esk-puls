package job

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

const JobsDirPath = "jobs"

func LoadJobsFromDir() ([]HttpJob, error) {
	var jobs []HttpJob

	if err := filepath.WalkDir(jobsDirPath(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !(filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".yaml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var job HttpJob
		if err := yaml.Unmarshal(data, &job); err != nil {
			return err
		}
		jobs = append(jobs, job)
		return nil
	}); err != nil {
		return nil, err
	}
	return jobs, nil
}

func projectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(b), "..", "..", "..")
	return filepath.Clean(root) + string(filepath.Separator)
}

func jobsDirPath() string {
	return filepath.Join(projectRoot(), JobsDirPath)
}
