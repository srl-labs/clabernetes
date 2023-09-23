package clabverter

type startupConfigConfigMapTemplateVars struct {
	Name          string
	Namespace     string
	StartupConfig string
}

type extraFilesConfigMapTemplateVars struct {
	Name       string
	Namespace  string
	ExtraFiles map[string]string
}

type topologyConfigMapTemplateVars struct {
	NodeName      string
	ConfigMapName string
	FilePath      string
	FileName      string
}

type containerlabTemplateVars struct {
	Name               string
	Namespace          string
	ClabConfig         string
	StartupConfigs     []topologyConfigMapTemplateVars
	ExtraFiles         []topologyConfigMapTemplateVars
	InsecureRegistries []string
}

type renderedContent struct {
	friendlyName string
	fileName     string
	content      []byte
}

type sourceDestinationPathPair struct {
	sourcePath      string
	destinationPath string
}
