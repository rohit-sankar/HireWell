package ai

import "fmt"

// NewProvider is the single place AI_PROVIDER is read and dispatched. This
// is what makes "the AI provider must be replaceable later" a one-line
// change: add a case here, implement Provider, done. No other package
// branches on provider name.
func NewProvider(providerName, apiKey, model string) (Provider, error) {
	switch providerName {
	case "", "stub":
		return NewStubProvider(), nil
	case "gemini":
		if apiKey == "" {
			return nil, fmt.Errorf("AI_PROVIDER=gemini requires AI_API_KEY to be set")
		}
		return NewGeminiProvider(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unknown AI_PROVIDER %q (supported: stub, gemini)", providerName)
	}
}
