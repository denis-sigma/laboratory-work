package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(
	baseURL string,
	timeout time.Duration,
) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type request struct {
	RequestID   string            `json:"request_id"`
	RecipientID uuid.UUID         `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

func (c *Client) RequestPosition(
	ctx context.Context,
	driverID uuid.UUID,
	tripID uuid.UUID,
) error {
	ctx, span := otel.Tracer("tripgo/push").Start(ctx, "push.request_position")
	defer span.End()
	span.SetAttributes(
		attribute.String("trip.id", tripID.String()),
		attribute.String("driver.id", driverID.String()),
	)
	payload, err := json.Marshal(request{
		RequestID:   uuid.NewString(),
		RecipientID: driverID,
		Kind:        "REQUEST_POSITION",
		Data: map[string]string{
			"trip_id": tripID.String(),
		},
	})
	if err != nil {
		return fmt.Errorf("сериализация push-запроса: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v1/push",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("создание push-запроса: %w", err)
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("вызов Push Service: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)

		//nolint:staticcheck // preserve the external service name in the error.
		return fmt.Errorf(
			"Push Service вернул HTTP %d",
			response.StatusCode,
		)
	}

	_, _ = io.Copy(io.Discard, response.Body)

	return nil
}
