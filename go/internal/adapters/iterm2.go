package adapters

type SessionLocator struct {
	Host        string
	Application string
	SessionID   string
}

type FocusRequest struct {
	Locator SessionLocator
}

type FocusResult struct {
	Supported bool
	Reason    string
}

type FocusAdapter interface {
	Focus(request FocusRequest) (FocusResult, error)
}

type Iterm2Adapter struct{}

func (a Iterm2Adapter) Focus(request FocusRequest) (FocusResult, error) {
	_ = request
	return FocusResult{
		Supported: false,
		Reason:    "iTerm2 focus automation is deferred; adapter boundary is bootstrapped.",
	}, nil
}
