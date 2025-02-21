package resources

type ResourceTester interface {
	GetName() string
	CreateResource(string, chan<- error) error
	ReadResource(string, chan<- error) error
	DeleteResource(string, chan<- error) error
}
