package launcher

import (
	"fmt"
	"os"
	"path/filepath"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	"gopkg.in/yaml.v3"
)

func (c *clabernetes) getFilesFromURL() error {
	content, err := os.ReadFile("files-from-url.yaml")
	if err != nil {
		return err
	}

	var filesFromURL []clabernetesapistopologyv1alpha1.FileFromURL

	err = yaml.Unmarshal(content, &filesFromURL)
	if err != nil {
		return err
	}

	for _, fileFromURL := range filesFromURL {
		var f *os.File

		err = os.MkdirAll(
			filepath.Dir(fileFromURL.FilePath),
			clabernetesconstants.PermissionsEveryoneReadWrite,
		)
		if err != nil {
			return err
		}

		f, err = os.Create(fmt.Sprintf("/clabernetes/%s", fileFromURL.FilePath))
		if err != nil {
			return err
		}

		err = clabernetesutil.WriteHTTPContentsFromPath(
			c.ctx,
			fileFromURL.URL,
			f,
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
