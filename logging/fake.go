package logging

var _ Instance = (*FakeInstance)(nil)

// FakeInstance is a fake logging instance that does nothing.
type FakeInstance struct{}

func (i *FakeInstance) Debug(f string) {}

func (i *FakeInstance) Debugf(f string, a ...interface{}) {}

func (i *FakeInstance) Info(f string) {}

func (i *FakeInstance) Infof(f string, a ...interface{}) {}

func (i *FakeInstance) Warn(f string) {}

func (i *FakeInstance) Warnf(f string, a ...interface{}) {}

func (i *FakeInstance) Critical(f string) {}

func (i *FakeInstance) Criticalf(f string, a ...interface{}) {}

func (i *FakeInstance) Fatal(f string) {}

func (i *FakeInstance) Fatalf(f string, a ...interface{}) {}

func (i *FakeInstance) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (i *FakeInstance) GetName() string {
	return ""
}

func (i *FakeInstance) GetLevel() string {
	return ""
}
