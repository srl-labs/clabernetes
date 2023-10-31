package launcher

import (
	"fmt"
	"os"

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

		// TODO -- does os.Create create the path structure if it doesnt exist? prolly no? test that

		f, err = os.Create(fmt.Sprintf("/clabernetes/%s", fileFromURL.FilePath))
		if err != nil {
			return err
		}

		err = clabernetesutil.WriteHTTPContentsFromPath(
			fileFromURL.URL,
			f,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
