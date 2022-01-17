package plugin

var defaultImpl Plugin = &plugin{}

type Opts struct {
}

type Plugin interface {
	Register(opts *Opts) error
}

func Register(opts *Opts) error {
	return defaultImpl.Register(opts)
}

type plugin struct {
}

func (p *plugin) Register(opts *Opts) error {
	return nil
}
