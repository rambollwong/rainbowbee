package tcp

import "context"

type Option func(n *Network) error

// apply load configuration items for Network instance.
func (n *Network) apply(opt ...Option) error {
	for _, o := range opt {
		if err := o(n); err != nil {
			return err
		}
	}
	return nil
}

func WithContext(ctx context.Context) Option {
	return func(n *Network) error {
		n.ctx = ctx
		return nil
	}
}
