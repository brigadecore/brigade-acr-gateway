package webhooks

import (
	"context"

	"github.com/brigadecore/brigade/sdk/v3"
	"github.com/pkg/errors"
)

// Service is an interface for components that can handle webhooks (events) from
// ACR. Implementations of this interface are transport-agnostic.
type Service interface {
	// Handle handles a webhook (event) from ACR.
	Handle(ctx context.Context, event Event, payload []byte) error
}

type service struct {
	eventsClient sdk.EventsClient
}

// NewService returns an implementation of the Service interface for handling
// webhooks (events) from ACR.
func NewService(eventsClient sdk.EventsClient) Service {
	return &service{
		eventsClient: eventsClient,
	}
}

func (s *service) Handle(
	ctx context.Context,
	event Event,
	payload []byte,
) error {
	switch event.Action {
	case "ping":
		return nil
	case "push", "delete":
	default:
		return errors.Errorf(
			"received event with unsupported action %q",
			event.Action,
		)
	}
	brigadeEvent := sdk.Event{
		Source: "brigade.sh/acr",
		Type:   event.Action,
		Qualifiers: map[string]string{
			"registry": event.Request.Host,
		},
		Labels: map[string]string{
			"repo": event.Target.Repository,
		},
		Payload: string(payload),
	}
	_, err := s.eventsClient.Create(ctx, brigadeEvent, nil)
	return errors.Wrap(err, "error emitting event(s) into Brigade")
}
